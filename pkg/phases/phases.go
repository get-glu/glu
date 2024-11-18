package phases

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
)

// Source is an interface around storage for resources.
type Source[R core.Resource] interface {
	Type() string
	View(_ context.Context, pipeline, phase core.Metadata, _ R) error
}

type UpdatableSource[R core.Resource] interface {
	Source[R]
	Update(_ context.Context, pipeline, phase core.Metadata, from, to R) error
}

// Pipeline is a set of phase with promotion dependencies between one another.
type Pipeline[R core.Resource] interface {
	New() R
	Metadata() core.Metadata
	Add(r core.ResourcePhase[R], opts ...containers.Option[core.AddPhaseOptions[R]]) error
	PromotedFrom(core.ResourcePhase[R]) (core.ResourcePhase[R], bool)
}

type Phase[R core.Resource] struct {
	logger   *slog.Logger
	meta     core.Metadata
	pipeline Pipeline[R]
	source   Source[R]
}

func New[R core.Resource](meta core.Metadata, pipeline Pipeline[R], repo Source[R], opts ...containers.Option[core.AddPhaseOptions[R]]) (*Phase[R], error) {
	logger := slog.With("name", meta.Name, "pipeline", pipeline.Metadata().Name)
	for k, v := range meta.Labels {
		logger = logger.With(k, v)
	}

	phase := &Phase[R]{
		logger:   logger,
		meta:     meta,
		pipeline: pipeline,
		source:   repo,
	}

	if err := pipeline.Add(phase, opts...); err != nil {
		return nil, err
	}

	return phase, nil
}

func (i *Phase[R]) Metadata() core.Metadata {
	return i.meta
}

func (i *Phase[R]) SourceType() string {
	return i.source.Type()
}

func (i *Phase[R]) Get(ctx context.Context) (any, error) {
	return i.GetResource(ctx)
}

// GetResource returns the identified resource as its concrete pointer type.
func (i *Phase[R]) GetResource(ctx context.Context) (a R, err error) {
	a = i.pipeline.New()
	if err := i.source.View(ctx, i.pipeline.Metadata(), i.meta, a); err != nil {
		return a, err
	}

	return a, nil
}

// Promote causes the phase to attempt a promotion from a dependent phase.
// If there is no promotion phase, this process is skipped.
// The phase fetches both its current resource state, and that of the promotion source phase.
// If the resources differ, then the phase updates its source to match the promoted version.
func (i *Phase[R]) Promote(ctx context.Context) (err error) {
	i.logger.Debug("Promotion started")
	defer func() {
		i.logger.Debug("Promotion finished")
		if err != nil {
			err = fmt.Errorf("promoting %s/%s: %w", i.pipeline.Metadata().Name, i.meta.Name, err)
		}
	}()

	updatable, ok := i.source.(UpdatableSource[R])
	if !ok {
		return nil
	}

	from := i.pipeline.New()
	if err := i.source.View(ctx, i.pipeline.Metadata(), i.meta, from); err != nil {
		return err
	}

	dep, ok := i.pipeline.PromotedFrom(i)
	if !ok {
		return nil
	}

	to, err := dep.GetResource(ctx)
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
		i.logger.Debug("skipping promotion", "reason", "UpToDate")

		return nil
	}

	if err := updatable.Update(ctx, i.pipeline.Metadata(), i.meta, from, to); err != nil {
		return fmt.Errorf("updating from %q to %q: %w", fromDigest, toDigest, err)
	}

	return nil
}
