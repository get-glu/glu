package builder

import (
	"github.com/get-glu/glu"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/phases"
	srcgit "github.com/get-glu/glu/pkg/src/git"
	srcoci "github.com/get-glu/glu/pkg/src/oci"
)

// SystemBuilder is a utility type for populating systems with type constrained pipelines.
// It has a number of utilities for simplifying common configuration options used
// to create types sources and phases within a typed pipeline.
type SystemBuilder[R glu.Resource] struct {
	*glu.System

	err error
}

// New constructs and configures a new system builder.
func New[R glu.Resource](system *glu.System) *SystemBuilder[R] {
	return &SystemBuilder[R]{System: system}
}

type PipelineBuilder[R glu.Resource] interface {
	Configuration() (*glu.Config, error)
	// NewPhase constructs a new phase and registers it on the builders resulting pipeline.
	NewPhase(meta glu.Metadata, source phases.Source[R], _ ...containers.Option[phases.Options[R]]) (*phases.Phase[R], error)
}

// AddTrigger delegates to the underlying system but returns the system builder.
func (b *SystemBuilder[R]) AddTrigger(t glu.Trigger) *SystemBuilder[R] {
	b.System.AddTrigger(t)
	return b
}

// Run delegates to the underlying system Run method after checking for any
// previously observed errors.
func (b *SystemBuilder[R]) Run() error {
	if b.err != nil {
		return b.err
	}

	return b.System.Run()
}

// BuildPipeline creates a new pipeline and invokes the provided build function with a pipeline builder.
// It is up to the caller to use this builder in order to create new sources and phases.
func (b *SystemBuilder[R]) BuildPipeline(meta glu.Metadata, newFunc func() R, build func(builder PipelineBuilder[R]) error) *SystemBuilder[R] {
	// skip if we previously observed an error
	if b.err != nil {
		return b
	}

	pipeline := glu.NewPipeline[R](meta)

	if err := build(pipelineBuilder[R]{b, pipeline, newFunc}); err != nil {
		b.err = err
		return b
	}

	b.System.AddPipeline(pipeline)

	return b
}

type pipelineBuilder[R glu.Resource] struct {
	*SystemBuilder[R]

	pipeline *glu.ResourcePipeline[R]
	newFn    func() R
}

// NewPhase constructs a new phase and registers it on the builders resulting pipeline.
func (p pipelineBuilder[R]) NewPhase(meta glu.Metadata, source phases.Source[R], opts ...containers.Option[phases.Options[R]]) (*phases.Phase[R], error) {
	return phases.New(meta, p.pipeline, source, p.newFn, opts...)
}

// GitSource is a convenience function for building a git.Source implementation using a pipeline builder implementation.
func GitSource[R srcgit.Resource](builder PipelineBuilder[R], name string, opts ...containers.Option[srcgit.Source[R]]) (*srcgit.Source[R], error) {
	config, err := builder.Configuration()
	if err != nil {
		return nil, err
	}

	repo, proposer, err := config.GitRepository(name)
	if err != nil {
		return nil, err
	}

	return srcgit.NewSource(repo, proposer, opts...), nil
}

// OCISource  is a convenience function for building an oci.Source implementation using a pipeline builder implementation.
func OCISource[R srcoci.Resource](builder PipelineBuilder[R], name string, opts ...containers.Option[srcoci.Source[R]]) (*srcoci.Source[R], error) {
	config, err := builder.Configuration()
	if err != nil {
		return nil, err
	}

	repo, err := config.OCIRepository(name)
	if err != nil {
		return nil, err
	}

	return srcoci.New[R](repo), nil
}
