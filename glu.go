package glu

import (
	"context"
	"reflect"

	"github.com/flipt-io/glu/pkg/fs"
)

type Repository interface {
	View(context.Context, func(fs.Filesystem) error) error
	Update(context.Context, func(fs.Filesystem) error) error
}

type Pipeline struct {
	ctx context.Context
	src Repository
}

func NewPipeline(ctx context.Context) *Pipeline {
	return &Pipeline{ctx: ctx}
}

func (p *Pipeline) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

type Phase struct {
	pipeline *Pipeline
	name     string
	src      Repository
}

func (p *Phase) Name() string {
	return p.name
}

func (p *Pipeline) NewPhase(name string) *Phase {
	return &Phase{
		pipeline: p,
		name:     name,
		src:      p.src,
	}
}

type Metadata struct {
	Name   string
	Labels map[string]string
}

type App interface {
	Metadata() Metadata
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
}](phase *Phase, p P, opts ...InstanceOption[A, P]) *Instance[A, P] {
	inst := Instance[A, P]{phase: phase}
	for _, opt := range opts {
		opt(&inst)
	}

	return &inst
}

func (i *Instance[A, P]) Reconcile(ctx context.Context) (P, error) {
	a := P(new(A))
	if err := i.phase.src.View(ctx, func(f fs.Filesystem) error {
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
		return a, nil
	}

	if err := i.phase.src.Update(ctx, func(f fs.Filesystem) error {
		return b.WriteTo(ctx, i.phase, f)
	}); err != nil {
		return nil, err
	}

	return b, nil
}
