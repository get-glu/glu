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

type Phase = core.Phase

type Pipeline struct {
	*core.Pipeline

	ctx   context.Context
	conf  *config.Config
	creds *credentials.CredentialSource
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
	Reconcile(context.Context) (A, error)
}

type Instance[A any, P interface {
	*A
	core.Resource
}] struct {
	phase *core.Phase
	meta  core.Metadata
	fn    func(core.Metadata) P
	src   Reconciler[P]
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
}](phase *core.Phase, meta core.Metadata, fn func(core.Metadata) P, opts ...InstanceOption[A, P]) *Instance[A, P] {
	inst := Instance[A, P]{phase: phase, meta: meta, fn: fn}
	for _, opt := range opts {
		opt(&inst)
	}

	return &inst
}

func (i *Instance[A, P]) Reconcile(ctx context.Context) (P, error) {
	slog.Debug("reconcile started", "type", "instance", "phase", i.phase.Name(), "name", i.meta.Name)

	repo := i.phase.Repository()

	a := i.fn(i.meta)
	if err := repo.View(ctx, i.phase, func(f fs.Filesystem) error {
		return a.ReadFrom(ctx, i.phase, f)
	}); err != nil {
		return nil, err
	}

	if i.src == nil {
		return a, nil
	}

	b, err := i.src.Reconcile(ctx)
	if err != nil {
		return nil, err
	}

	aDigest, err := a.Digest()
	if err != nil {
		return nil, err
	}

	bDigest, err := b.Digest()
	if err != nil {
		return nil, err
	}

	if aDigest == bDigest {
		slog.Debug("skipping reconcile", "reason", "UpToDate")

		return a, nil
	}

	if err := repo.Update(ctx, i.phase, a, b, func(f fs.Filesystem) (string, error) {
		return fmt.Sprintf("Update %s in %s", i.meta.Name, i.phase.Name()), b.WriteTo(ctx, i.phase, f)
	}); err != nil {
		return nil, err
	}

	return b, nil
}
