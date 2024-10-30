package glu

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/flipt-io/glu/pkg/config"
	"github.com/flipt-io/glu/pkg/credentials"
	"github.com/flipt-io/glu/pkg/fs"
)

type Pipeline struct {
	ctx   context.Context
	name  string
	conf  *config.Config
	creds *credentials.CredentialSource
}

func NewPipeline(ctx context.Context, name string) (*Pipeline, error) {
	// TODO(georgemac): make glu config path optionally configurable
	conf, err := config.ReadFromPath("glu.yaml")
	if err != nil {
		return nil, err
	}

	return &Pipeline{
		ctx:   ctx,
		name:  name,
		conf:  conf,
		creds: credentials.New(conf.Credentials),
	}, nil
}

func (p *Pipeline) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

type Phase struct {
	pipeline *Pipeline
	name     string
	// TODO(georgemac): make optionall configurable
	branch string
	repo   Repository
}

func (p *Phase) Name() string {
	return p.name
}

func (p *Phase) Branch() string {
	return p.branch
}

func (p *Pipeline) NewPhase(name string, repo Repository) *Phase {
	return &Phase{
		pipeline: p,
		name:     name,
		branch:   "main",
		repo:     repo,
	}
}

type Metadata struct {
	Name   string
	Labels map[string]string
}

type App interface {
	ReadFrom(context.Context, *Phase, fs.Filesystem) error
	WriteTo(_ context.Context, _ *Phase, _ fs.Filesystem) error
}

type Reconciler[A any] interface {
	Reconcile(context.Context) (A, error)
}

type Instance[A any, P interface {
	*A
	App
}] struct {
	phase *Phase
	meta  Metadata
	fn    func(Metadata) P
	src   Reconciler[P]
}

type InstanceOption[A any, P interface {
	*A
	App
}] func(*Instance[A, P])

func DependsOn[A any, P interface {
	*A
	App
}](src Reconciler[P]) InstanceOption[A, P] {
	return func(i *Instance[A, P]) {
		i.src = src
	}
}

func NewInstance[A any, P interface {
	*A
	App
}](phase *Phase, meta Metadata, fn func(Metadata) P, opts ...InstanceOption[A, P]) *Instance[A, P] {
	inst := Instance[A, P]{phase: phase, meta: meta, fn: fn}
	for _, opt := range opts {
		opt(&inst)
	}

	return &inst
}

func (i *Instance[A, P]) Reconcile(ctx context.Context) (P, error) {
	slog.Debug("reconcile started", "type", "instance", "phase", i.phase.name, "name", i.meta.Name)

	a := i.fn(i.meta)
	if err := i.phase.repo.View(ctx, i.phase, func(f fs.Filesystem) error {
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

	if reflect.DeepEqual(a, b) {
		slog.Debug("skipping reconcile", "reason", "UpToDate")
		return a, nil
	}

	if err := i.phase.repo.Update(ctx, i.phase, &i.meta, func(f fs.Filesystem) (string, error) {
		return fmt.Sprintf("Update %s in %s", i.meta.Name, i.phase.name), b.WriteTo(ctx, i.phase, f)
	}); err != nil {
		return nil, err
	}

	return b, nil
}
