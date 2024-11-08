package glu

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/credentials"
	"github.com/get-glu/glu/pkg/src/git"
	"github.com/get-glu/glu/pkg/src/oci"
	"golang.org/x/sync/errgroup"
)

const defaultScheduleInternal = time.Minute

var ErrNotFound = errors.New("not found")

type Metadata = core.Metadata

type Resource = core.Resource

func Name(name string) Metadata {
	return Metadata{Name: name}
}

func NewPipeline[R core.Resource](meta Metadata, newFn func() R) *core.Pipeline[R] {
	return core.NewPipeline(meta, newFn)
}

type Pipeline interface {
	Metadata() Metadata
	Controllers() map[string]core.Controller
	Dependencies() map[core.Controller]core.Controller
}

type System struct {
	conf      *config.Config
	pipelines map[string]Pipeline
	schedules []Schedule
	err       error

	server *Server
}

func NewSystem() *System {
	r := &System{
		pipelines: map[string]Pipeline{},
	}

	r.server = newServer(r)

	return r
}

type Scheduler interface {
	ScheduleReconcile(opts ...containers.Option[Schedule])
}

func (s *System) AddPipeline(fn func(Config, Scheduler) (Pipeline, error)) *System {
	config, err := s.configuration()
	if err != nil {
		s.err = err
		return s
	}

	pipe, err := fn(config, s)
	if err != nil {
		s.err = err
		return s
	}

	s.pipelines[pipe.Metadata().Name] = pipe
	return s
}

func (s *System) configuration() (_ Config, err error) {
	if s.conf != nil {
		return newConfigSource(s.conf), nil
	}

	s.conf, err = config.ReadFromPath("glu.yaml")
	if err != nil {
		return nil, err
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(s.conf.Log.Level)); err != nil {
		return nil, err
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))

	return newConfigSource(s.conf), nil
}

func (s *System) Run(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) > 1 {
		return s.runOnce(ctx)
	}

	var (
		group errgroup.Group
		srv   = http.Server{
			Addr:    ":8080", // TODO: make configurable
			Handler: s.server,
		}
	)

	group.Go(func() error {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})

	group.Go(func() error {
		slog.Info("starting server", "addr", ":8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	group.Go(func() error {
		return s.run(ctx)
	})

	return group.Wait()
}

func (s *System) runOnce(ctx context.Context) error {
	switch os.Args[1] {
	case "inspect":
		if len(os.Args) < 3 {
			return fmt.Errorf("inspect <kind> (expected kind argument (one of [pipeline phase]))")
		}

		// inspect <kind> ...
		switch os.Args[2] {
		case "pipeline":
			return s.inspectPipeline(ctx, os.Args[3:]...)
		default:
			return fmt.Errorf("unexpected kind: %q", os.Args[2])
		}
	case "reconcile":
		if len(os.Args) < 3 {
			return fmt.Errorf("reconcile <kind> (expected kind argument (one of [pipeline phase]))")
		}

		// reconcile <kind> ...
		switch os.Args[2] {
		case "pipeline":
			return s.reconcilePipeline(ctx, os.Args[2:]...)
		default:
			return fmt.Errorf("unexpected kind: %q", os.Args[2])
		}
	default:
		return fmt.Errorf("unexpected command %q (expected one of [inspect reconcile])", os.Args[1])
	}
}

func (s *System) getPipeline(name string) (Pipeline, error) {
	pipeline, ok := s.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("pipeline %q: %w", name, ErrNotFound)
	}

	return pipeline, nil
}

func (s *System) inspectPipeline(ctx context.Context, args ...string) (err error) {
	wr := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer func() {
		if ferr := wr.Flush(); ferr != nil && err == nil {
			err = ferr
		}
	}()

	if len(args) == 0 {
		fmt.Fprintln(wr, "NAME")
		for name := range s.pipelines {
			fmt.Fprintln(wr, name)
		}
		return nil
	}

	pipeline, err := s.getPipeline(args[0])
	if err != nil {
		return err
	}

	if len(args) == 1 {
		fmt.Fprintln(wr, "NAME\tDEPENDS_ON")
		deps := pipeline.Dependencies()
		for name, controller := range pipeline.Controllers() {
			dependsName := ""
			if depends, ok := deps[controller]; ok && depends != nil {
				dependsName = depends.Metadata().Name
			}

			fmt.Fprintf(wr, "%s\t%s\n", name, dependsName)
		}
		return nil
	}

	controller, ok := pipeline.Controllers()[args[1]]
	if !ok {
		return fmt.Errorf("controller not found: %q", args[1])
	}

	inst, err := controller.Get(ctx)
	if err != nil {
		return err
	}

	var extraFields [][2]string
	fields, ok := inst.(fields)
	if ok {
		extraFields = fields.PrinterFields()
	}

	fmt.Fprint(wr, "NAME")
	for _, field := range extraFields {
		fmt.Fprintf(wr, "\t%s", field[0])
	}
	fmt.Fprintln(wr)

	meta := controller.Metadata()
	fmt.Fprintf(wr, "%s", meta.Name)
	for _, field := range extraFields {
		fmt.Fprintf(wr, "\t%s", field[1])
	}
	fmt.Fprintln(wr)

	return nil
}

type fields interface {
	PrinterFields() [][2]string
}

func (s *System) reconcilePipeline(ctx context.Context, args ...string) error {
	if len(args) < 2 {
		return fmt.Errorf("reconcile <pipeline> <controller>")
	}

	pipeline, err := s.getPipeline(args[0])
	if err != nil {
		return err
	}

	controller, ok := pipeline.Controllers()[args[1]]
	if !ok {
		return fmt.Errorf("controller not found: %q", args[1])
	}

	if err := controller.Reconcile(ctx); err != nil {
		return err
	}

	return nil
}

type Schedule struct {
	interval          time.Duration
	matchesController core.Controller
	matchesLabels     map[string]string
}

func (s Schedule) matches(c core.Controller) bool {
	if s.matchesController != nil {
		return s.matchesController == c
	}

	if len(s.matchesLabels) > 0 {
		labels := c.Metadata().Labels
		for k, v := range s.matchesLabels {
			if found, ok := labels[k]; !ok || v != found {
				return false
			}
		}
	}

	return true
}

func (s *System) ScheduleReconcile(opts ...containers.Option[Schedule]) {
	sch := Schedule{
		interval: defaultScheduleInternal,
	}

	containers.ApplyAll(&sch, opts...)

	s.schedules = append(s.schedules, sch)
}

func ScheduleInterval(d time.Duration) containers.Option[Schedule] {
	return func(s *Schedule) {
		s.interval = d
	}
}

func ScheduleMatchesController(c core.Controller) containers.Option[Schedule] {
	return func(s *Schedule) {
		s.matchesController = c
	}
}

func ScheduleMatchesLabel(k, v string) containers.Option[Schedule] {
	return func(s *Schedule) {
		if s.matchesLabels == nil {
			s.matchesLabels = map[string]string{}
		}

		s.matchesLabels[k] = v
	}
}

func (s *System) run(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, sch := range s.schedules {
		wg.Add(1)
		go func(sch Schedule) {
			defer wg.Done()

			ticker := time.NewTicker(sch.interval)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					for _, pipeline := range s.pipelines {
						for _, controller := range pipeline.Controllers() {
							if sch.matches(controller) {
								if err := controller.Reconcile(ctx); err != nil {
									slog.Error("reconciling resource", "name", controller.Metadata().Name, "error", err)
								}
							}
						}
					}

				}
			}
		}(sch)
	}

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		wg.Wait()
	}()

	<-ctx.Done()

	select {
	case <-time.After(15 * time.Second):
		return errors.New("timedout waiting on shutdown of schedules")
	case <-finished:
		return ctx.Err()
	}
}

type Config interface {
	git.ConfigSource
	oci.ConfigSource
}

type configSource struct {
	conf  *config.Config
	creds *credentials.CredentialSource
}

func newConfigSource(conf *config.Config) *configSource {
	return &configSource{
		conf:  conf,
		creds: credentials.New(conf.Credentials),
	}
}

func (c *configSource) GitRepositoryConfig(name string) (*config.Repository, error) {
	conf, ok := c.conf.Sources.Git[name]
	if !ok {
		return nil, fmt.Errorf("git %q: configuration not found", name)
	}

	return conf, nil
}

func (c *configSource) OCIRepositoryConfig(name string) (*config.OCIRepository, error) {
	conf, ok := c.conf.Sources.OCI[name]
	if !ok {
		return nil, fmt.Errorf("oci %q: configuration not found", name)
	}

	return conf, nil
}

func (c *configSource) GetCredential(name string) (*credentials.Credential, error) {
	return c.creds.Get(name)
}
