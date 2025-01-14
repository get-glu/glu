package core

import (
	"context"
	"fmt"
	"iter"
	"maps"

	"github.com/get-glu/glu/pkg/containers"
)

// System is the primary entrypoint for build a set of Glu pipelines.
type System struct {
	ctx       context.Context
	pipelines map[string]*Pipeline
	err       error
}

// NewSystem constructs and configures a new system.
func NewSystem(ctx context.Context, opts ...containers.Option[System]) *System {
	r := &System{
		ctx:       ctx,
		pipelines: map[string]*Pipeline{},
	}

	containers.ApplyAll(r, opts...)

	return r
}

// Context returns the systems root context.
func (s *System) Context() context.Context {
	return s.ctx
}

// GetPipeline returns a pipeline by name.
func (s *System) GetPipeline(name string) (*Pipeline, error) {
	pipeline, ok := s.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("pipeline %q: %w", name, ErrNotFound)
	}

	return pipeline, nil
}

// Pipelines returns an iterator across all name and pipeline pairs
// previously registered on the system.
func (s *System) Pipelines() iter.Seq2[string, *Pipeline] {
	return maps.All(s.pipelines)
}

// AddPipeline invokes a pipeline builder function provided by the caller.
// The function is provided with the systems configuration and (if successful)
// the system registers the resulting pipeline.
func (s *System) AddPipeline(pipeline *Pipeline) *System {
	s.pipelines[pipeline.Metadata().Name] = pipeline

	return s
}
