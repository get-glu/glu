package glu

import (
	"context"
	"errors"
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

	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/credentials"
	"github.com/get-glu/glu/pkg/repository"
	githubscm "github.com/get-glu/glu/pkg/scm/github"
	"github.com/get-glu/glu/pkg/sources/git"
	giturls "github.com/whilp/git-urls"
	"golang.org/x/sync/errgroup"
)

var ErrNotFound = errors.New("not found")

type Metadata = core.Metadata

var DefaultRegistry = NewRegistry()

type Registry struct {
	conf      *config.Config
	pipelines map[string]*Pipeline

	server *Server
}

func NewRegistry() *Registry {
	r := &Registry{
		pipelines: map[string]*Pipeline{},
	}

	r.server = newServer(r)
	return r
}

func (r *Registry) getConf() (_ *config.Config, err error) {
	if r.conf != nil {
		return r.conf, nil
	}

	r.conf, err = config.ReadFromPath("glu.yaml")
	if err != nil {
		return nil, err
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(r.conf.Log.Level)); err != nil {
		return nil, err
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))

	return r.conf, nil
}

func Run(ctx context.Context) error {
	return DefaultRegistry.Run(ctx)
}

func (r *Registry) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) > 1 {
		return r.runOnce(ctx)
	}

	var (
		group errgroup.Group
		srv   = http.Server{
			Addr:    ":8080", // TODO: make configurable
			Handler: r.server,
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

	for _, p := range r.pipelines {
		group.Go(func() error {
			return p.run(ctx)
		})
	}

	return group.Wait()
}

func (r *Registry) runOnce(ctx context.Context) error {
	switch os.Args[1] {
	case "inspect":
		// inspect [pipeline] [phase] [resource]
		return r.inspect(ctx, os.Args[2:]...)
	case "reconcile":
		// reconcile <pipeline> <phase> <resource>
		return r.reconcile(ctx, os.Args[2:]...)
	default:
		return fmt.Errorf("unexpected command %q (expected one of [inspect reconcile])", os.Args[1])
	}
}

func (r *Registry) getPipeline(name string) (*Pipeline, error) {
	pipeline, ok := r.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("pipeline %q: %w", name, ErrNotFound)
	}

	return pipeline, nil
}

func (r *Registry) inspect(ctx context.Context, args ...string) (err error) {
	wr := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer func() {
		if ferr := wr.Flush(); ferr != nil && err == nil {
			err = ferr
		}
	}()

	if len(args) == 0 {
		fmt.Fprintln(wr, "PIPELINES")
		for name := range r.pipelines {
			fmt.Fprintln(wr, name)
		}
		return nil
	}

	pipeline, err := r.getPipeline(args[0])
	if err != nil {
		return err
	}

	if len(args) == 1 {
		fmt.Fprintln(wr, "PHASES")
		for name := range pipeline.Phases() {
			fmt.Fprintln(wr, name)
		}
		return nil
	}

	phase, err := pipeline.getPhase(args[1])
	if err != nil {
		return err
	}

	if len(args) == 2 {
		fmt.Fprintln(wr, "RESOURCES")
		for name := range phase {
			fmt.Fprintln(wr, name)
		}
		return nil
	}

	reconciler, err := getResource(phase, args[2])
	if err != nil {
		return err
	}

	inst, err := reconciler.Get(ctx)
	if err != nil {
		return err
	}

	var extraFields [][2]string
	fields, ok := inst.(fields)
	if ok {
		extraFields = fields.PrinterFields()
	}

	fmt.Fprint(wr, "PHASE\tRESOURCE")
	for _, field := range extraFields {
		fmt.Fprintf(wr, "\t%s", field[0])
	}
	fmt.Fprintln(wr)

	meta := reconciler.Metadata()
	fmt.Fprintf(wr, "%s\t%s", meta.Phase, meta.Name)
	for _, field := range extraFields {
		fmt.Fprintf(wr, "\t%s", field[1])
	}
	fmt.Fprintln(wr)

	return nil
}

type fields interface {
	PrinterFields() [][2]string
}

func (r *Registry) reconcile(ctx context.Context, args ...string) error {
	if len(args) < 2 {
		return fmt.Errorf("reconcile <pipeline> <phase> [resource]")
	}

	pipeline, err := r.getPipeline(args[0])
	if err != nil {
		return err
	}

	phase, err := pipeline.getPhase(args[1])
	if err != nil {
		return err
	}

	if len(args) < 3 {
		// attempt to reconcile everything in the current phase
		for _, reconciler := range phase {
			if err := reconciler.Reconcile(ctx); err != nil {
				return err
			}
		}

		return nil
	}

	resource, err := getResource(phase, args[2])
	if err != nil {
		return err
	}

	return resource.Reconcile(ctx)
}

func NewPipeline(ctx context.Context, name string) (*Pipeline, error) {
	return DefaultRegistry.NewPipeline(ctx, name)
}

func (r *Registry) NewPipeline(ctx context.Context, name string) (*Pipeline, error) {
	conf, err := r.getConf()
	if err != nil {
		return nil, err
	}

	pipeline := newPipeline(ctx, conf, name)

	r.pipelines[name] = pipeline

	return pipeline, nil
}

type Pipeline struct {
	*core.Pipeline

	ctx   context.Context
	conf  *config.Config
	creds *credentials.CredentialSource

	scheduled []scheduled
}

type scheduled struct {
	core.Reconciler

	interval time.Duration
}

func newPipeline(ctx context.Context, conf *config.Config, name string) *Pipeline {
	return &Pipeline{
		Pipeline: core.NewPipeline(ctx, name),
		ctx:      ctx,
		conf:     conf,
		creds:    credentials.New(conf.Credentials),
	}
}

func (p *Pipeline) getPhase(name string) (map[string]core.Reconciler, error) {
	m, ok := p.Phases()[name]
	if !ok {
		return nil, fmt.Errorf(`phase "%q/%q": %w`, p.Name(), name, ErrNotFound)
	}

	return m, nil
}

func getResource(phase map[string]core.Reconciler, name string) (core.Reconciler, error) {
	reconciler, ok := phase[name]
	if !ok {
		return nil, fmt.Errorf(`resource %q: %w`, name, ErrNotFound)
	}

	return reconciler, nil
}

func (p *Pipeline) run(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, sch := range p.scheduled {
		wg.Add(1)
		go func(sch scheduled) {
			defer wg.Done()

			ticker := time.NewTicker(sch.interval)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := sch.Reconcile(ctx); err != nil {
						meta := sch.Metadata()
						slog.Error("reconciling resource", "phase", meta.Phase, "name", meta.Name, "error", err)
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

func (p *Pipeline) ScheduleReconcile(r core.Reconciler, interval time.Duration) {
	p.scheduled = append(p.scheduled, scheduled{r, interval})
}

type RepositoryOptions struct {
	enableProposals bool
}

func EnableProposals(o *RepositoryOptions) {
	o.enableProposals = true
}

func (p *Pipeline) NewRepository(name string, opts ...containers.Option[RepositoryOptions]) (git.Repository, error) {
	var options RepositoryOptions
	containers.ApplyAll(&options, opts...)

	conf, ok := p.conf.Repositories[name]
	if !ok {
		return nil, fmt.Errorf("repository %q: configuration not found", name)
	}

	repoOpts := []containers.Option[repository.GitRepository]{}
	if conf.Proposals != nil {
		repoURL, err := giturls.Parse(conf.Remote.URL)
		if err != nil {
			return nil, err
		}

		parts := strings.SplitN(strings.TrimPrefix(repoURL.Path, "/"), "/", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("unexpected repository URL path: %q", repoURL.Path)
		}

		var (
			repoOwner = parts[0]
			repoName  = strings.TrimSuffix(parts[1], ".git")
		)

		var proposalsEnabled bool
		if proposalsEnabled = conf.Proposals.Credential != ""; proposalsEnabled {
			creds, err := p.creds.Get(conf.Proposals.Credential)
			if err != nil {
				return nil, fmt.Errorf("repository %q: %w", name, err)
			}

			client, err := creds.GitHubClient(p.ctx)
			if err != nil {
				return nil, fmt.Errorf("repository %q: %w", name, err)
			}

			repoOpts = append(repoOpts, repository.WithProposer(githubscm.New(
				client.PullRequests,
				repoOwner,
				repoName,
			)))
		}

		slog.Debug("configured scm proposer",
			slog.String("owner", repoOwner),
			slog.String("name", repoName),
			slog.Bool("proposals_enabled", proposalsEnabled),
		)
	}

	return repository.NewGitRepository(
		p.ctx,
		conf,
		p.creds,
		name,
		repoOpts...,
	)
}
