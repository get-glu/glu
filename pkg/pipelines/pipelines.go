package pipelines

import (
	"context"

	"github.com/get-glu/glu"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core/typed"
	"github.com/get-glu/glu/pkg/edges"
	srcgit "github.com/get-glu/glu/pkg/phases/git"
	srcoci "github.com/get-glu/glu/pkg/phases/oci"
	"github.com/get-glu/glu/pkg/triggers"
)

var _ Builder[glu.Resource] = (*PipelineBuilder[glu.Resource])(nil)

// PipelineBuilder is a utility type for populating systems with type constrained pipelines.
// It has a number of utilities for simplifying common configuration options used
// to create types sources and phases within a typed pipeline.
type PipelineBuilder[R glu.Resource] struct {
	pipeline *glu.Pipeline

	system *glu.System
	config *glu.Config
	newFn  func() R

	err error
}

func (p *PipelineBuilder[R]) New() R {
	return p.newFn()
}

func (p *PipelineBuilder[R]) PipelineName() string {
	return p.pipeline.Metadata().Name
}

func (p *PipelineBuilder[R]) Configuration() *glu.Config {
	return p.config
}

func (p *PipelineBuilder[R]) Context() context.Context {
	return p.system.Context()
}

// NewBuilder constructs and configures a new pipeline builder.
func NewBuilder[R glu.Resource](system *glu.System, meta glu.Metadata, newFn func() R) *PipelineBuilder[R] {
	config, err := system.Configuration()
	if err != nil {
		return &PipelineBuilder[R]{err: err}
	}

	return &PipelineBuilder[R]{
		system:   system,
		config:   config,
		pipeline: glu.NewPipeline(meta),
		newFn:    newFn,
	}
}

func (b *PipelineBuilder[R]) Build() error {
	if b.err != nil {
		return b.err
	}

	b.system.AddPipeline(b.pipeline)

	return nil
}

// NewPhase constructs a new phase and registers it on the resulting pipeline produced by the builder.
func (b *PipelineBuilder[R]) NewPhase(fn func(b Builder[R]) (typed.Phase[R], error)) (next *PhaseBuilder[R]) {
	next = &PhaseBuilder[R]{PipelineBuilder: b}
	if b.err != nil {
		return
	}

	phase, err := fn(b)
	if err != nil {
		b.err = err
		return
	}

	if err := b.pipeline.AddPhase(phase); err != nil {
		b.err = err
		return
	}

	next.phase = phase

	return
}

// PhaseBuilder is used to chain building new phases with edges leading to the same phase.
type PhaseBuilder[R glu.Resource] struct {
	*PipelineBuilder[R]
	phase typed.Phase[R]
}

// PromotesTo creates a new phase and an edge to this new phase from the phase built in the receiver.
func (b *PhaseBuilder[R]) PromotesTo(fn func(b Builder[R]) (typed.UpdatablePhase[R], error), ts ...triggers.Trigger) (next *PhaseBuilder[R]) {
	next = &PhaseBuilder[R]{PipelineBuilder: b.PipelineBuilder}
	if b.err != nil {
		return
	}

	to, err := fn(b)
	if err != nil {
		b.err = err
		return
	}

	if err := b.pipeline.AddPhase(to); err != nil {
		b.err = err
		return
	}

	if err := b.pipeline.AddEdge(triggers.Edge(edges.Promotes(b.phase, to), ts...)); err != nil {
		b.err = err
		return
	}

	next.phase = to

	return
}

// Builder is used carry dependencies for building new phases
type Builder[R glu.Resource] interface {
	Context() context.Context
	Configuration() *glu.Config
	PipelineName() string
	New() R
}

// GitPhase is a convenience function for building a git.Phase implementation using a pipeline builder implementation.
func GitPhase[R srcgit.Resource](builder Builder[R], meta glu.Metadata, srcName string, opts ...containers.Option[srcgit.Phase[R]]) (*srcgit.Phase[R], error) {
	repo, proposer, err := builder.Configuration().GitRepository(srcName)
	if err != nil {
		return nil, err
	}

	ctx := builder.Context()

	phase, err := srcgit.New(
		ctx,
		builder.PipelineName(),
		meta,
		builder.New,
		repo,
		proposer,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	return phase, nil
}

// OCIPhase is a convenience function for building an oci.Phase implementation using a pipeline builder implementation.
func OCIPhase[R srcoci.Resource](builder Builder[R], meta glu.Metadata, srcName string, opts ...containers.Option[srcoci.Phase[R]]) (*srcoci.Phase[R], error) {
	repo, err := builder.Configuration().OCIRepository(srcName)
	if err != nil {
		return nil, err
	}

	phase := srcoci.New(
		builder.PipelineName(),
		meta,
		builder.New,
		repo,
		opts...,
	)

	return phase, nil
}
