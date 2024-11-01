package glu

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/flipt-io/glu/pkg/config"
	"github.com/flipt-io/glu/pkg/containers"
	"github.com/flipt-io/glu/pkg/core"
	"github.com/flipt-io/glu/pkg/credentials"
	"github.com/flipt-io/glu/pkg/fs"
	"github.com/flipt-io/glu/pkg/repository"
)

type Metadata = core.Metadata

type Pipeline struct {
	*core.Pipeline

	ctx   context.Context
	conf  *config.Config
	creds *credentials.CredentialSource

	reconcilers []reconciler
}

type reconciler interface {
	metadata() core.Metadata
	Reconcile(context.Context) error
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

type Reconciler[A any] interface {
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
	src  Reconciler[P]
}

type InstanceOption[A any, P interface {
	*A
	core.Resource
}] func(*Instance[A, P])

func DependsOn[A any, P interface {
	*A
	core.Resource
}](src Reconciler[P]) InstanceOption[A, P] {
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

func (i *Instance[A, P]) metadata() core.Metadata {
	return i.meta
}

func (i *Instance[A, P]) Get(ctx context.Context) (P, error) {
	p := i.fn(i.meta)
	if err := i.repo.View(ctx, p, func(f fs.Filesystem) error {
		return p.ReadFrom(ctx, f)
	}); err != nil {
		return nil, err
	}

	return p, nil
}

func (i *Instance[A, P]) Reconcile(ctx context.Context) error {
	slog.Debug("reconcile started", "type", "instance", "phase", i.meta.Phase, "name", i.meta.Name)

	from := i.fn(i.meta)
	if err := i.repo.View(ctx, from, func(f fs.Filesystem) error {
		return from.ReadFrom(ctx, f)
	}); err != nil {
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

	if err := i.repo.Update(ctx, from, to, func(f fs.Filesystem) (string, error) {
		return fmt.Sprintf("Update %s in %s", i.meta.Name, i.meta.Phase), to.WriteTo(ctx, f)
	}); err != nil {
		return err
	}

	return nil
}
