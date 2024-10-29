package glu

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"reflect"

	"github.com/flipt-io/glu/pkg/fs"
)

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
	repo     Repository
}

func (p *Phase) Name() string {
	return p.name
}

func (p *Pipeline) NewPhase(name string, repo Repository) *Phase {
	return &Phase{
		pipeline: p,
		name:     name,
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
	slog.Debug("Reconcile", "type", "instance", "name", i.meta.Name)

	a := i.fn(i.meta)
	if err := i.phase.repo.View(ctx, func(f fs.Filesystem) error {
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

	branch := fmt.Sprintf("glu/reconcile/%s/%s/%x", i.meta.Name, i.phase.name, rand.Int63())

	if err := i.phase.repo.Update(ctx, branch, func(f fs.Filesystem) (string, error) {
		return fmt.Sprintf("Update %s in %s", i.meta.Name, i.phase.name), b.WriteTo(ctx, i.phase, f)
	}); err != nil {
		return nil, err
	}

	return b, nil
}
