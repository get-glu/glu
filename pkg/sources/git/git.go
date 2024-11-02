package git

import (
	"context"
	"log/slog"

	"github.com/flipt-io/glu/pkg/core"
)

type Source[A any] interface {
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
	src  Source[P]
}

type InstanceOption[A any, P interface {
	*A
	core.Resource
}] func(*Instance[A, P])

func DependsOn[A any, P interface {
	*A
	core.Resource
}](src Source[P]) InstanceOption[A, P] {
	return func(i *Instance[A, P]) {
		i.src = src
	}
}

type Registry interface {
	Register(core.Reconciler)
}

func New[A any, P interface {
	*A
	core.Resource
}](
	pipeline Registry,
	repo core.Repository,
	meta core.Metadata,
	fn func(core.Metadata) P,
	opts ...InstanceOption[A, P]) *Instance[A, P] {

	inst := &Instance[A, P]{repo: repo, meta: meta, fn: fn}
	for _, opt := range opts {
		opt(inst)
	}

	pipeline.Register(inst)

	return inst
}

func (i *Instance[A, P]) Metadata() core.Metadata {
	return i.meta
}

func (i *Instance[A, P]) GetAny(ctx context.Context) (any, error) {
	return i.Get(ctx)
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
