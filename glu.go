package glu

import (
	"context"
	"log/slog"

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

type Source[A any] interface {
	Subscribe(context.Context, chan<- A)
}

type Instance[A any, P interface {
	*A
	App
}] struct {
	phase *Phase
	src   Source[A]
}

type InstanceOption[A any, P interface {
	*A
	App
}] func(*Instance[A, P])

func DerivedFrom[A any, P interface {
	*A
	App
}](src Source[A]) InstanceOption[A, P] {
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

	if inst.src != nil {
		ctx := phase.pipeline.ctx

		ch := make(chan A)
		inst.src.Subscribe(ctx, ch)

		go func() {
			select {
			case <-ctx.Done():
				return
			case a := <-ch:
				if err := inst.reconcile(ctx, &a); err != nil {
					slog.Error("reconciling instance", "error", err)
				}
			}
		}()
	}
	return &inst
}

func (i *Instance[A, P]) Get(ctx context.Context) (*A, error) {
	a := P(new(A))
	if err := i.phase.src.View(ctx, func(f fs.Filesystem) error {
		return a.ReadFrom(ctx, i.phase, f)
	}); err != nil {
		return nil, err
	}

	return a, nil
}

func (i *Instance[A, P]) reconcile(ctx context.Context, p P) error {
	return i.phase.src.Update(ctx, func(f fs.Filesystem) error {
		return p.WriteTo(ctx, i.phase, f)
	})
}
