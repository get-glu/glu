package glu

import (
	"fmt"
	"iter"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/phases"
)

// Pipeline is an alias for the core Pipeline interface (see core.Pipeline)
type Pipeline = core.Pipeline

// Phase is an alias for the core Phase interface (see core.Phase)
type Phase = core.Phase

type entry[R Resource] struct {
	core.ResourcePhase[R]
	opts phases.Options[R]
}

// ResourcePipeline is a collection of phases for a given resource type R.
// It implements the core.Phase interface and is scoped to a single Resource implementation.
type ResourcePipeline[R Resource] struct {
	meta  Metadata
	newFn func() R
	nodes map[string]entry[R]
}

// NewPipeline constructs and configures a new instance of *ResourcePipeline[R]
func NewPipeline[R Resource](meta Metadata) *ResourcePipeline[R] {
	return &ResourcePipeline[R]{
		meta:  meta,
		nodes: map[string]entry[R]{},
	}
}

// Metadata returns the metadata assocated with the Pipelines (name and labels).
func (p *ResourcePipeline[R]) Metadata() Metadata {
	return p.meta
}

// Add will add the provided resource phase to the pipeline along with configuring
// any dependent promotion source phases if configured to do so.
func (p *ResourcePipeline[R]) Add(r core.ResourcePhase[R], opts ...containers.Option[phases.Options[R]]) error {
	add := phases.Options[R]{}
	containers.ApplyAll(&add, opts...)

	if _, existing := p.nodes[r.Metadata().Name]; existing {
		return fmt.Errorf("phase %q: %w", r.Metadata().Name, core.ErrAlreadyExists)
	}

	p.nodes[r.Metadata().Name] = entry[R]{r, add}

	return nil
}

// PromotedFrom returns the phase which c is configured to promote from (get dependent phase).
func (p *ResourcePipeline[R]) PromotedFrom(c core.ResourcePhase[R]) (core.ResourcePhase[R], bool) {
	entry, ok := p.nodes[c.Metadata().Name]
	if !ok || entry.opts.PromotedFrom == nil {
		return nil, false
	}

	return entry.opts.PromotedFrom, true
}

// PhaseByName returns the Phase (if it exists) with a matching name.
func (p *ResourcePipeline[R]) PhaseByName(name string) (Phase, error) {
	entry, ok := p.nodes[name]
	if !ok {
		return nil, fmt.Errorf("phase %q: %w", name, core.ErrNotFound)
	}

	return entry.ResourcePhase, nil
}

// Phases lists all phases in the pipeline with optional predicates.
func (p *ResourcePipeline[R]) Phases(opts ...containers.Option[core.PhaseOptions]) iter.Seq[Phase] {
	var options core.PhaseOptions
	containers.ApplyAll(&options, opts...)

	return iter.Seq[Phase](func(yield func(Phase) bool) {
		for _, entry := range p.nodes {
			if !options.Matches(entry.ResourcePhase) {
				continue
			}

			if !yield(entry.ResourcePhase) {
				break
			}
		}
	})
}

// Dependencies returns a map of Phase to Phase.
// This map contains mappings of Phases to their dependent promotion source Phase (if configured).
func (p *ResourcePipeline[R]) Dependencies() map[Phase]Phase {
	deps := map[Phase]Phase{}
	for _, entry := range p.nodes {
		deps[entry.ResourcePhase] = entry.opts.PromotedFrom
	}

	return deps
}
