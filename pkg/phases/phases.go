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
	Metadata() core.Metadata
	View(_ context.Context, pipeline, phase core.Metadata, _ R) error
	Subscribe(pipeline, phase core.Metadata, newFn func() R, record func(R, map[string]string))
}

// UpdatableSource is a source through which the phase can promote resources to new versions
type UpdatableSource[R core.Resource] interface {
	Source[R]
	Update(_ context.Context, pipeline, phase core.Metadata, from, to R) (map[string]string, error)
}

// Pipeline is a set of phase with promotion dependencies between one another.
type Pipeline[R core.Resource] interface {
	Metadata() core.Metadata
	Add(r core.ResourcePhase[R], opts ...containers.Option[Options[R]]) error
	PromotedFrom(core.ResourcePhase[R]) (core.ResourcePhase[R], bool)
}

// Options are used to configure the addition of a ResourcePhase to a Pipeline
type Options[R core.Resource] struct {
	PromotedFrom core.ResourcePhase[R]
	Log          RefLog[R]
}

// PromotesFrom configures a dependent Phase to promote from for the Phase being added.
func PromotesFrom[R core.Resource](c core.ResourcePhase[R]) containers.Option[Options[R]] {
	return func(o *Options[R]) {
		o.PromotedFrom = c
	}
}

// LogsTo adds a reference log to the phase.
func LogsTo[R core.Resource](l RefLog[R]) containers.Option[Options[R]] {
	return func(o *Options[R]) {
		o.Log = l
	}
}

type RefLog[R core.Resource] interface {
	CreateReference(ctx context.Context, pipeline, phase core.Metadata) error
	RecordLatest(ctx context.Context, pipeline, phase core.Metadata, resource R, annotations map[string]string) error
	History(ctx context.Context, pipeline, phase core.Metadata) ([]core.State, error)
}

type Phase[R core.Resource] struct {
	Options[R]

	logger   *slog.Logger
	meta     core.Metadata
	pipeline Pipeline[R]
	source   Source[R]
	newFn    func() R
}

func New[R core.Resource](meta core.Metadata, pipeline Pipeline[R], source Source[R], newFn func() R, opts ...containers.Option[Options[R]]) (*Phase[R], error) {
	logger := slog.With("phase", meta.Name, "pipeline", pipeline.Metadata().Name)
	for k, v := range meta.Labels {
		logger = logger.With(k, v)
	}

	phase := &Phase[R]{
		logger:   logger,
		meta:     meta,
		pipeline: pipeline,
		source:   source,
		newFn:    newFn,
	}

	containers.ApplyAll(&phase.Options, opts...)

	if err := pipeline.Add(phase, opts...); err != nil {
		return nil, err
	}

	if phase.Log != nil {
		ctx := context.Background()
		if err := phase.Log.CreateReference(ctx, pipeline.Metadata(), phase.meta); err != nil {
			return nil, err
		}

		source.Subscribe(pipeline.Metadata(), phase.meta, newFn, func(r R, m map[string]string) {
			if err := phase.Log.RecordLatest(ctx, pipeline.Metadata(), phase.meta, r, m); err != nil {
				logger.Error("recording latest ref", "error", err)
			}
		})
	}

	return phase, nil
}

func (i *Phase[R]) Metadata() core.Metadata {
	return i.meta
}

func (i *Phase[R]) Source() core.Metadata {
	return i.source.Metadata()
}

func (i *Phase[R]) Get(ctx context.Context) (core.Resource, error) {
	return i.GetResource(ctx)
}

// GetResource returns the identified resource as its concrete pointer type.
func (i *Phase[R]) GetResource(ctx context.Context) (a R, err error) {
	a = i.newFn()
	if err := i.source.View(ctx, i.pipeline.Metadata(), i.meta, a); err != nil {
		return a, err
	}

	return a, nil
}

// Promote causes the phase to attempt a promotion from a dependent phase.
// If there is no promotion phase, this process is skipped.
// The phase fetches both its current resource state, and that of the promotion source phase.
// If the resources differ, then the phase updates its source to match the promoted version.
func (i *Phase[R]) Promote(ctx context.Context) (r core.PromotionResult, err error) {
	i.logger.Debug("Promotion started")
	defer func() {
		i.logger.Debug("Promotion finished")
		if err != nil {
			err = fmt.Errorf("promoting %s/%s: %w", i.pipeline.Metadata().Name, i.meta.Name, err)
		}
	}()

	updatable, ok := i.source.(UpdatableSource[R])
	if !ok {
		return r, nil
	}

	from, to, synced, err := i.synced(ctx)
	if err != nil {
		return r, err
	}

	if synced {
		i.logger.Debug("skipping promotion", "reason", "UpToDate")
		return r, nil
	}

	if r.Annotations, err = updatable.Update(ctx, i.pipeline.Metadata(), i.meta, from, to); err != nil {
		return r, err
	}

	return r, nil
}

func (i *Phase[R]) Synced(ctx context.Context) (bool, error) {
	_, _, synced, err := i.synced(ctx)
	return synced, err
}

func (i *Phase[R]) synced(ctx context.Context) (from, to R, synced bool, err error) {
	from = i.newFn()
	if err := i.source.View(ctx, i.pipeline.Metadata(), i.meta, from); err != nil {
		return from, to, false, err
	}

	dep, ok := i.pipeline.PromotedFrom(i)
	if !ok {
		return from, to, true, nil
	}

	to, err = dep.GetResource(ctx)
	if err != nil {
		return from, to, false, err
	}

	fromDigest, err := from.Digest()
	if err != nil {
		return from, to, false, err
	}

	toDigest, err := to.Digest()
	if err != nil {
		return from, to, false, err
	}

	if fromDigest == toDigest {
		return from, to, true, nil
	}

	return from, to, false, nil
}

func (i *Phase[R]) History(ctx context.Context) ([]core.State, error) {
	if i.Options.Log == nil {
		return nil, nil
	}

	return i.Options.Log.History(ctx, i.pipeline.Metadata(), i.meta)
}
