package pipelines

import (
	"github.com/get-glu/glu"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/edges"
	srcgit "github.com/get-glu/glu/pkg/phases/git"
	srcoci "github.com/get-glu/glu/pkg/phases/oci"
)

var _ Builder[glu.Resource] = (*PipelineBuilder[glu.Resource])(nil)

// PipelineBuilder is a utility type for populating systems with type constrained pipelines.
// It has a number of utilities for simplifying common configuration options used
// to create types sources and phases within a typed pipeline.
type PipelineBuilder[R glu.Resource] struct {
	pipeline *glu.Pipeline
	config   *glu.Config
	newFn    func() R

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

// NewBuilder constructs and configures a new pipeline builder.
func NewBuilder[R glu.Resource](config *glu.Config, meta glu.Metadata, newFn func() R) *PipelineBuilder[R] {
	pipeline := glu.NewPipeline(meta)
	return &PipelineBuilder[R]{config: config, pipeline: pipeline, newFn: newFn}
}

func (b *PipelineBuilder[R]) Build(system *glu.System) error {
	if b.err != nil {
		return b.err
	}

	system.AddPipeline(b.pipeline)

	return nil
}

func (b *PipelineBuilder[R]) NewPhase(fn func(b Builder[R]) (edges.Phase[R], error)) (next *PhaseBuilder[R]) {
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

type PhaseBuilder[R glu.Resource] struct {
	*PipelineBuilder[R]
	phase edges.Phase[R]
}

func (b *PhaseBuilder[R]) PromotesTo(fn func(b Builder[R]) (edges.UpdatablePhase[R], error)) (next *PhaseBuilder[R]) {
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

type Builder[R glu.Resource] interface {
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

	phase := srcgit.NewPhase(
		builder.PipelineName(),
		meta,
		builder.New,
		repo,
		proposer,
		opts...,
	)

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
	)

	return phase, nil
}
