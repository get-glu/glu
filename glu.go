package glu

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/flipt-io/glu/pkg/config"
	"github.com/flipt-io/glu/pkg/containers"
	"github.com/flipt-io/glu/pkg/core"
	"github.com/flipt-io/glu/pkg/credentials"
	"github.com/flipt-io/glu/pkg/repository"
)

type Metadata = core.Metadata

type Pipeline struct {
	*core.Pipeline

	ctx   context.Context
	conf  *config.Config
	creds *credentials.CredentialSource

	reconcilers []Reconciler
	scheduled   []scheduled
}

type Reconciler interface {
	Metadata() core.Metadata
	Reconcile(context.Context) error
}

type scheduled struct {
	Reconciler

	interval time.Duration
}

func NewPipeline(ctx context.Context, name string) (*Pipeline, error) {
	conf, err := config.ReadFromPath("glu.yaml")
	if err != nil {
		return nil, err
	}

	return &Pipeline{
		Pipeline: core.NewPipeline(ctx, name),
		ctx:      ctx,
		conf:     conf,
		creds:    credentials.New(conf.Credentials),
	}, nil
}

func (p *Pipeline) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

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

func (p *Pipeline) ScheduleReconcile(r Reconciler, interval time.Duration) {
	p.scheduled = append(p.scheduled, scheduled{r, interval})
}

type RepositoryOptions struct {
	enableProposals bool
}

func EnableProposals(o *RepositoryOptions) {
	o.enableProposals = true
}

func (p *Pipeline) NewRepository(name string, opts ...containers.Option[RepositoryOptions]) (core.Repository, error) {
	var options RepositoryOptions
	containers.ApplyAll(&options, opts...)

	conf, ok := p.conf.Repositories[name]
	if !ok {
		return nil, fmt.Errorf("repository %q: configuration not found", name)
	}

	return repository.NewGitRepository(p.ctx, conf, p.creds, name, options.enableProposals)
}

type GetReconciler[A any] interface {
	Get(context.Context) (A, error)
	Reconcile(context.Context) error
}

type Instance[A any, P interface {
	*A
	core.Resource
}] struct {
	repo core.Repository
	meta core.Metadata
	fn   func(core.Metadata) P
	src  GetReconciler[P]
}

type InstanceOption[A any, P interface {
	*A
	core.Resource
}] func(*Instance[A, P])

func DependsOn[A any, P interface {
	*A
	core.Resource
}](src GetReconciler[P]) InstanceOption[A, P] {
	return func(i *Instance[A, P]) {
		i.src = src
	}
}

func NewInstance[A any, P interface {
	*A
	core.Resource
}](
	pipeline *Pipeline,
	repo core.Repository,
	meta core.Metadata,
	fn func(core.Metadata) P,
	opts ...InstanceOption[A, P]) *Instance[A, P] {

	inst := &Instance[A, P]{repo: repo, meta: meta, fn: fn}
	for _, opt := range opts {
		opt(inst)
	}

	pipeline.reconcilers = append(pipeline.reconcilers, inst)

	return inst
}

func (i *Instance[A, P]) Metadata() core.Metadata {
	return i.meta
}

func (i *Instance[A, P]) Get(ctx context.Context) (P, error) {
	p := i.fn(i.meta)
	if err := i.repo.View(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

func (i *Instance[A, P]) Reconcile(ctx context.Context) error {
	slog.Debug("reconcile started", "type", "instance", "phase", i.meta.Phase, "name", i.meta.Name)

	from := i.fn(i.meta)
	if err := i.repo.View(ctx, from); err != nil {
		return err
	}

	if i.src == nil {
		// nothing to reconcile from
		return nil
	}

	if err := i.src.Reconcile(ctx); err != nil {
		return err
	}

	to, err := i.src.Get(ctx)
	if err != nil {
		return err
	}

	fromDigest, err := from.Digest()
	if err != nil {
		return err
	}

	toDigest, err := to.Digest()
	if err != nil {
		return err
	}

	if fromDigest == toDigest {
		slog.Debug("skipping reconcile", "reason", "UpToDate")

		return nil
	}

	if err := i.repo.Update(ctx, from, to); err != nil {
		return err
	}

	return nil
}
