package pipelines

import (
	"context"

	"github.com/get-glu/glu"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/edges"
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
func NewBuilder[R glu.Resource](system *glu.System, meta glu.Metadata, newFn func() R, opts ...containers.Option[PipelineBuilder[R]]) *PipelineBuilder[R] {
	config, err := system.Configuration()
	if err != nil {
		return &PipelineBuilder[R]{err: err}
	}

	builder := &PipelineBuilder[R]{
		system:   system,
		config:   config,
		pipeline: glu.NewPipeline(meta),
		newFn:    newFn,
	}

	containers.ApplyAll(builder, opts...)

	return builder
}

func (b *PipelineBuilder[R]) Build() error {
	if b.err != nil {
		return b.err
	}

	b.system.AddPipeline(b.pipeline)

	return nil
}

// NewPhase constructs a new phase and registers it on the resulting pipeline produced by the builder.
func (b *PipelineBuilder[R]) NewPhase(fn func(b Builder[R]) (core.Phase, error)) (next *PhaseBuilder[R]) {
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
	phase core.Phase
}

// PromotesTo creates a new phase and an edge to this new phase from the phase built in the receiver.
func (b *PhaseBuilder[R]) PromotesTo(fn func(b Builder[R]) (core.Phase, error)) (next *PhaseBuilder[R]) {
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

	if err := b.pipeline.AddEdge(edges.Promotes(b.phase, to)); err != nil {
		b.err = err
		return
	}

	next.phase = to

	return
}

// Builder is used carry dependencies for building new phases
type Builder[R glu.Resource] interface {
	New() R
	Context() context.Context
	PipelineName() string
	Configuration() *glu.Config
}
