package controllers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
)

// Source is an interface around storage for resources.
type Source[R core.Resource] interface {
	View(_ context.Context, pipeline, controller core.Metadata, _ R) error
}

type UpdatableSource[R core.Resource] interface {
	Source[R]
	Update(_ context.Context, pipeline, controller core.Metadata, from, to R) error
}

// Pipeline is a set of reconcilers organised into phases.
type Pipeline[R core.Resource] interface {
	New() R
	Metadata() core.Metadata
	Add(r core.ResourceController[R], opts ...containers.Option[core.AddOptions[R]])
	GetDependency(core.ResourceController[R]) (core.ResourceController[R], bool)
}

type Controller[R core.Resource] struct {
	logger   *slog.Logger
	meta     core.Metadata
	pipeline Pipeline[R]
	source   Source[R]
}

func New[R core.Resource](meta core.Metadata, pipeline Pipeline[R], repo Source[R], opts ...containers.Option[core.AddOptions[R]]) *Controller[R] {
	logger := slog.With("name", meta.Name, "pipeline", pipeline.Metadata().Name)
	for k, v := range meta.Labels {
		logger = logger.With(k, v)
	}

	controller := &Controller[R]{
		logger:   logger,
		meta:     meta,
		pipeline: pipeline,
		source:   repo,
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
	if err := i.source.View(ctx, i.pipeline.Metadata(), i.meta, a); err != nil {
		return a, err
	}

	return a, nil
}

// Reconcile forces the controller to retrieve the latest version of the resouce from the underlying repository.
// If a dependent controller has been defined, then it is also reconciled.
// If the dependent controller resource differs, then the controller attempts
// to update its underlying repository to match the new desired state.
func (i *Controller[R]) Reconcile(ctx context.Context) (err error) {
	i.logger.Debug("Reconcile started")
	defer func() {
		i.logger.Debug("Reconcile finished")
		if err != nil {
			err = fmt.Errorf("reconciling %s/%s: %w", i.pipeline.Metadata().Name, i.meta.Name, err)
		}
	}()

	from := i.pipeline.New()
	if err := i.source.View(ctx, i.pipeline.Metadata(), i.meta, from); err != nil {
		return err
	}

	src, ok := i.pipeline.GetDependency(i)
	if !ok {
		return nil
	}

	updatable, ok := i.source.(UpdatableSource[R])
	if !ok {
		return nil
	}

	if err := src.Reconcile(ctx); err != nil {
		return fmt.Errorf("reconciling source dependency %q: %w", src.Metadata().Name, err)
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
		i.logger.Debug("skipping reconcile", "reason", "UpToDate")

		return nil
	}

	if err := updatable.Update(ctx, i.pipeline.Metadata(), i.meta, from, to); err != nil {
		return fmt.Errorf("updating from %q to %q: %w", fromDigest, toDigest, err)
	}

	return nil
}
