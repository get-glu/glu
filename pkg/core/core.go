package core

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/get-glu/glu/pkg/containers"
)

var (
	// ErrNotFound is returned when a particular resource cannot be located
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned when an attempt is made to create a resource
	// which already exists
	ErrAlreadyExists = errors.New("already exists")
)

// Metadata contains the unique information used to identify
// a named resource instance in a particular phase.
type Metadata struct {
	Name   string
	Labels map[string]string
}

// Phase is the core interface for resource sourcing and management.
// These types can be registered on pipelines and can depend upon on another for promotion.
type Phase interface {
	Metadata() Metadata
	Get(context.Context) (any, error)
	Promote(context.Context) error
}

// Resource is an instance of a resource in a phase.
// Primarilly, it exposes a Digest method used to produce
// a hash digest of the resource instances current state.
type Resource interface {
	Digest() (string, error)
}

// Pipeline is a collection of phases for a given resource type R.
type Pipeline[R Resource] struct {
	meta  Metadata
	newFn func() R
	nodes map[string]entry[R]
}

type entry[R Resource] struct {
	ResourcePhase[R]
	promotesFrom ResourcePhase[R]
}

// NewPipeline constructs and configures a new instance of Pipeline
func NewPipeline[R Resource](meta Metadata, newFn func() R) *Pipeline[R] {
	return &Pipeline[R]{
		meta:  meta,
		newFn: newFn,
		nodes: map[string]entry[R]{},
	}
}

// AddOptions are used to configure the addition of a ResourcePhase to a Pipeline
type AddOptions[R Resource] struct {
	entry entry[R]
}

// PromotesFrom configures a dependent Phase to promote from for the Phase being added.
func PromotesFrom[R Resource](c ResourcePhase[R]) containers.Option[AddOptions[R]] {
	return func(ao *AddOptions[R]) {
		ao.entry.promotesFrom = c
	}
}

// ResourcePhase is a Phase bound to a particular resource type R.
type ResourcePhase[R Resource] interface {
	Phase
	GetResource(context.Context) (R, error)
}

// New calls the functions underlying resource constructor function to get a
// new default instance of the resource.
func (p *Pipeline[R]) New() R {
	return p.newFn()
}

// Metadata returns the metadata assocated with the Pipelines (name and labels).
func (p *Pipeline[R]) Metadata() Metadata {
	return p.meta
}

// Add will add the provided resource controller to the pipeline along with configuring
// any dependent promotion source phases if configured to do so.
func (p *Pipeline[R]) Add(r ResourcePhase[R], opts ...containers.Option[AddOptions[R]]) error {
	add := AddOptions[R]{entry: entry[R]{ResourcePhase: r}}
	containers.ApplyAll(&add, opts...)

	if _, existing := p.nodes[r.Metadata().Name]; existing {
		return fmt.Errorf("phase %q: %w", r.Metadata().Name, ErrAlreadyExists)
	}

	p.nodes[r.Metadata().Name] = add.entry

	return nil
}

// PromotedFrom returns the phase which c is configured to promote from (get dependent phase).
func (p *Pipeline[R]) PromotedFrom(c ResourcePhase[R]) (ResourcePhase[R], bool) {
	dep, ok := p.nodes[c.Metadata().Name]
	if !ok || dep.promotesFrom == nil {
		return nil, false
	}

	return dep.promotesFrom, true
}

// PhaseOptions scopes a call to get phases from a pipeline.
type PhaseOptions struct {
	phase  Phase
	labels map[string]string
}

// IsPhase causes a call to Phases to list specifically the provided phase p.
func IsPhase(p Phase) containers.Option[PhaseOptions] {
	return func(co *PhaseOptions) {
		co.phase = p
	}
}

// HasLabel causes a call to Phases to list any phase with the matching label paid k and v.
func HasLabel(k, v string) containers.Option[PhaseOptions] {
	return func(co *PhaseOptions) {
		if co.labels == nil {
			co.labels = map[string]string{}
		}

		co.labels[k] = v
	}
}

// HasAllLabels causes a call to Phases to list any phase which mataches all the provided labels.
func HasAllLabels(labels map[string]string) containers.Option[PhaseOptions] {
	return func(co *PhaseOptions) {
		co.labels = labels
	}
}

// PhaseByName returns the Phase (if it exists) with a matching name.
func (p *Pipeline[R]) PhaseByName(name string) (Phase, error) {
	entry, ok := p.nodes[name]
	if !ok {
		return nil, fmt.Errorf("phase %q: %w", name, ErrNotFound)
	}

	return entry.ResourcePhase, nil
}

// Phases lists all phases in the pipeline with optional predicates.
func (p *Pipeline[R]) Phases(opts ...containers.Option[PhaseOptions]) iter.Seq[Phase] {
	var options PhaseOptions
	containers.ApplyAll(&options, opts...)

	return iter.Seq[Phase](func(yield func(Phase) bool) {
		for _, entry := range p.nodes {
			if options.phase != nil && entry.ResourcePhase != options.phase {
				continue
			}

			if !hasAllLabels(entry.ResourcePhase, options.labels) {
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
func (p *Pipeline[R]) Dependencies() map[Phase]Phase {
	deps := map[Phase]Phase{}
	for _, entry := range p.nodes {
		deps[entry.ResourcePhase] = entry.promotesFrom
	}

	return deps
}

// hasAllLabels returns true if the provided phase has all the supplied labels
func hasAllLabels(c Phase, toFind map[string]string) bool {
	labels := c.Metadata().Labels
	for k, v := range toFind {
		if found, ok := labels[k]; !ok || v != found {
			return false
		}
	}
	return true
}
