package glu

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/get-glu/glu/internal/git"
	"github.com/get-glu/glu/internal/oci"
	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/credentials"
	"github.com/get-glu/glu/pkg/scm/github"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	giturls "github.com/whilp/git-urls"
	"golang.org/x/sync/errgroup"
)

const defaultScheduleInternal = time.Minute

var ErrNotFound = errors.New("not found")

type Metadata = core.Metadata

type Resource = core.Resource

func Name(name string, opts ...containers.Option[Metadata]) Metadata {
	meta := Metadata{Name: name}
	containers.ApplyAll(&meta, opts...)
	return meta
}

func Label(k, v string) containers.Option[Metadata] {
	return func(m *core.Metadata) {
		if m.Labels == nil {
			m.Labels = map[string]string{}
		}

		m.Labels[k] = v
	}
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
	ctx       context.Context
	conf      *config.Config
	pipelines map[string]Pipeline
	schedules []Schedule
	err       error

	server *Server
}

func NewSystem(ctx context.Context) *System {
	r := &System{
		ctx:       ctx,
		pipelines: map[string]Pipeline{},
	}

	r.server = newServer(r)

	return r
}

func (s *System) AddPipeline(fn func(context.Context, *Config) (Pipeline, error)) *System {
	config, err := s.configuration()
	if err != nil {
		s.err = err
		return s
	}

	pipe, err := fn(s.ctx, config)
	if err != nil {
		s.err = err
		return s
	}

	s.pipelines[pipe.Metadata().Name] = pipe
	return s
}

func (s *System) configuration() (_ *Config, err error) {
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

func (s *System) Run() error {
	if s.err != nil {
		return s.err
	}

	ctx, cancel := signal.NotifyContext(s.ctx, os.Interrupt, syscall.SIGTERM)
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
		return s.inspect(ctx, os.Args[2:]...)
	case "reconcile":
		return s.reconcile(ctx, os.Args[2:]...)
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

func (s *System) inspect(ctx context.Context, args ...string) (err error) {
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

type labels map[string]string

func (l labels) String() string {
	return "KEY=VALUE"
}

func (l labels) Set(v string) error {
	key, value, match := strings.Cut(v, "=")
	if !match {
		return fmt.Errorf("value should be in the form key=value (found %q)", v)
	}

	l[key] = value

	return nil
}

func (s *System) reconcile(ctx context.Context, args ...string) error {
	const usage = `reconcile <pipeline> FLAGS [controller <controller>]

    FLAGS:
    --label key=value`
	if len(args) < 1 {
		return errors.New(usage)
	}

	pipeline, err := s.getPipeline(args[0])
	if err != nil {
		return err
	}

	labels := labels{}
	set := flag.NewFlagSet("reconcile", flag.ExitOnError)
	set.Var(&labels, "label", "selector for filtering controllers (format key=value)")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}

	controllers := pipeline.Controllers()
	if set.NArg() < 1 {
		for _, controller := range controllers {
			if !contollerHasLabels(controller, labels) {
				continue
			}

			if err := controller.Reconcile(ctx); err != nil {
				return err
			}
		}
	}

	controller, ok := pipeline.Controllers()[set.Arg(0)]
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

	if !contollerHasLabels(c, s.matchesLabels) {
		return false
	}

	return true
}

func contollerHasLabels(c core.Controller, toFind map[string]string) bool {
	labels := c.Metadata().Labels
	for k, v := range toFind {
		if found, ok := labels[k]; !ok || v != found {
			return false
		}
	}
	return true
}

func (s *System) ScheduleReconcile(opts ...containers.Option[Schedule]) *System {
	sch := Schedule{
		interval: defaultScheduleInternal,
	}

	containers.ApplyAll(&sch, opts...)

	s.schedules = append(s.schedules, sch)

	return s
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

type Config struct {
	conf  *config.Config
	creds *credentials.CredentialSource

	cache struct {
		oci      map[string]*oci.Repository
		repo     map[string]*git.Repository
		proposer map[string]core.Proposer
	}
}

func newConfigSource(conf *config.Config) *Config {
	c := &Config{
		conf:  conf,
		creds: credentials.New(conf.Credentials),
	}

	c.cache.oci = map[string]*oci.Repository{}
	c.cache.repo = map[string]*git.Repository{}
	c.cache.proposer = map[string]core.Proposer{}

	return c
}

func (c *Config) GitRepository(ctx context.Context, name string) (_ *git.Repository, proposer core.Proposer, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("git %q: %w", name, err)
		}
	}()

	// check cache for previously built repository
	if repo, ok := c.cache.repo[name]; ok {
		return repo, c.cache.proposer[name], nil
	}

	conf, ok := c.conf.Sources.Git[name]
	if !ok {
		return nil, nil, errors.New("configuration not found")
	}

	var (
		method  transport.AuthMethod
		srcOpts = []containers.Option[git.Repository]{
			git.WithDefaultBranch(conf.DefaultBranch),
		}
	)

	if conf.Path != "" {
		srcOpts = append(srcOpts, git.WithFilesystemStorage(conf.Path))
	}

	if conf.Remote != nil {
		slog.Debug("configuring remote", "remote", conf.Remote.Name)

		srcOpts = append(srcOpts, git.WithRemote(conf.Remote.Name, conf.Remote.URL))

		if conf.Remote.Credential != "" {
			creds, err := c.creds.Get(conf.Remote.Credential)
			if err != nil {
				return nil, nil, fmt.Errorf("repository %q: %w", name, err)
			}

			method, err = creds.GitAuthentication()
			if err != nil {
				return nil, nil, fmt.Errorf("repository %q: %w", name, err)
			}
		}
	}

	if method == nil {
		method, err = ssh.DefaultAuthBuilder("git")
		if err != nil {
			return nil, nil, err
		}
	}

	repo, err := git.NewRepository(context.Background(), slog.Default(), append(srcOpts, git.WithAuth(method))...)
	if err != nil {
		return nil, nil, err
	}

	if conf.Proposals != nil {
		repoURL, err := giturls.Parse(conf.Remote.URL)
		if err != nil {
			return nil, nil, err
		}

		parts := strings.SplitN(strings.TrimPrefix(repoURL.Path, "/"), "/", 2)
		if len(parts) < 2 {
			return nil, nil, fmt.Errorf("unexpected repository URL path: %q", repoURL.Path)
		}

		var (
			repoOwner = parts[0]
			repoName  = strings.TrimSuffix(parts[1], ".git")
		)

		var proposalsEnabled bool
		if proposalsEnabled = conf.Proposals.Credential != ""; proposalsEnabled {
			creds, err := c.creds.Get(conf.Proposals.Credential)
			if err != nil {
				return nil, nil, err
			}

			client, err := creds.GitHubClient(ctx)
			if err != nil {
				return nil, nil, err
			}

			proposer = github.New(
				client.PullRequests,
				repoOwner,
				repoName,
			)
		}

		slog.Debug("configured scm proposer",
			slog.String("owner", repoOwner),
			slog.String("name", repoName),
			slog.Bool("proposals_enabled", proposalsEnabled),
		)
	}

	c.cache.repo[name] = repo
	c.cache.proposer[name] = proposer

	return repo, proposer, nil
}

func (c *Config) OCIRepository(name string) (_ *oci.Repository, err error) {
	// check cache for previously built repository
	if repo, ok := c.cache.oci[name]; ok {
		return repo, nil
	}

	conf, ok := c.conf.Sources.OCI[name]
	if !ok {
		return nil, fmt.Errorf("oci %q: configuration not found", name)
	}

	var cred *credentials.Credential
	if conf.Credential != "" {
		cred, err = c.creds.Get(conf.Credential)
		if err != nil {
			return nil, err
		}
	}

	repo, err := oci.New(conf.Reference, cred)
	if err != nil {
		return nil, err
	}

	c.cache.oci[name] = repo

	return repo, nil
}

func (c *Config) GetCredential(name string) (*credentials.Credential, error) {
	return c.creds.Get(name)
}
