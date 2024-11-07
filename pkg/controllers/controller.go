package controllers

import (
	"context"
	"log/slog"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
)

// Source is an interface around storage for resources.
type Source[R core.Resource] interface {
	View(context.Context, core.Metadata, R) error
}

type UpdatableSource[R core.Resource] interface {
	Source[R]
	Update(_ context.Context, _ core.Metadata, from, to R) error
}

// Pipeline is a set of reconcilers organised into phases.
type Pipeline[R core.Resource] interface {
	New() R
	Add(r core.ResourceController[R], opts ...containers.Option[core.AddOptions[R]])
	GetDependency(core.ResourceController[R]) (core.ResourceController[R], bool)
}

type Controller[R core.Resource] struct {
	meta     core.Metadata
	pipeline Pipeline[R]
	repo     Source[R]
}

func New[R core.Resource](meta core.Metadata, pipeline Pipeline[R], repo Source[R], opts ...containers.Option[core.AddOptions[R]]) *Controller[R] {
	controller := &Controller[R]{
		meta:     meta,
		pipeline: pipeline,
		repo:     repo,
	}

	pipeline.Add(controller, opts...)

	return controller
}

func (i *Controller[R]) Metadata() core.Metadata {
	return i.meta
}

func (i *Controller[R]) Get(ctx context.Context) (any, error) {
	return i.GetResource(ctx)
}

// GetResource returns the identified resource as its concrete pointer type.
func (i *Controller[R]) GetResource(ctx context.Context) (a R, err error) {
	a = i.pipeline.New()
	if err := i.repo.View(ctx, i.meta, a); err != nil {
		return a, err
	}

	return a, nil
}

// Reconcile forces the controller to retrieve the latest version of the resouce from the underlying repository.
// If a dependent controller has been defined, then it is also reconciled.
// If the dependent controller resource differs, then the controller attempts
// to update its underlying repository to match the new desired state.
func (i *Controller[R]) Reconcile(ctx context.Context) error {
	slog.Debug("reconcile started")

	from := i.pipeline.New()
	if err := i.repo.View(ctx, i.meta, from); err != nil {
		return err
	}

	src, ok := i.pipeline.GetDependency(i)
	if !ok {
		return nil
	}

	updatable, ok := i.repo.(UpdatableSource[R])
	if !ok {
		return nil
	}

	if err := src.Reconcile(ctx); err != nil {
		return err
	}

	to, err := src.GetResource(ctx)
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

	if err := updatable.Update(ctx, i.meta, from, to); err != nil {
		return err
	}

	return nil
}
