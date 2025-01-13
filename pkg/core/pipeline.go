package core

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/google/uuid"
)

// Phase is the core interface for resource sourcing and management.
// These types can be registered on pipelines and can depend upon on another for promotion.
type Phase interface {
	Descriptor() Descriptor
	Get(context.Context) (Resource, error)
}

// State contains a snapshot of a resource version at a point in history
type State struct {
	Version     uuid.UUID         `json:"version,omitempty"`
	Resource    Resource          `json:"resource,omitempty"`
	Digest      string            `json:"digest,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	RecordedAt  time.Time         `json:"recorded_at,omitempty"`
}

// Pipeline is a collection of phases for a given resource type R.
// It implements the core.Phase interface and is scoped to a single Resource implementation.
type Pipeline struct {
	meta   Metadata
	phases map[string]Phase
	edges  map[string]map[string]Edge
}

// NewPipeline constructs and configures a new instance of *ResourcePipeline[R]
func NewPipeline(meta Metadata) *Pipeline {
	return &Pipeline{
		meta:   meta,
		phases: map[string]Phase{},
		edges:  map[string]map[string]Edge{},
	}
}

// Metadata returns the metadata assocated with the Pipelines (name and labels).
func (p *Pipeline) Metadata() Metadata {
	return p.meta
}

// AddPhase will add the provided resource phase to the pipeline along with configuring
// any dependent promotion source phases if configured to do so.
func (p *Pipeline) AddPhase(phase Phase) error {
	name := phase.Descriptor().Metadata.Name
	if _, existing := p.phases[name]; existing {
		return fmt.Errorf("phase %q: %w", phase.Descriptor(), ErrAlreadyExists)
	}

	p.phases[name] = phase

	return nil
}

// PhaseByName returns the Phase (if it exists) with a matching name.
func (p *Pipeline) PhaseByName(name string) (Phase, error) {
	phase, ok := p.phases[name]
	if !ok {
		return nil, fmt.Errorf("phase %q: %w", name, ErrNotFound)
	}

	return phase, nil
}

// Phases lists all phases in the pipeline with optional predicates.
func (p *Pipeline) Phases(opts ...containers.Option[PhaseOptions]) iter.Seq[Phase] {
	var options PhaseOptions
	containers.ApplyAll(&options, opts...)

	return iter.Seq[Phase](func(yield func(Phase) bool) {
		for _, phase := range p.phases {
			if !options.Matches(phase) {
				continue
			}

			if !yield(phase) {
				break
			}
		}
	})
}

// Edge represents an edge between two phases.
// Edges have have their own kind which identifies their Perform behaviour.
type Edge interface {
	From() Descriptor
	To() Descriptor
}

// AddEdge adds an edge to a Pipeline.
func (p *Pipeline) AddEdge(e Edge) error {
	outgoing, ok := p.edges[e.From().Metadata.Name]
	if !ok {
		outgoing = map[string]Edge{}
		p.edges[e.From().Metadata.Name] = outgoing
	}

	if edge, ok := outgoing[e.To().Metadata.Name]; ok {
		return fmt.Errorf("edge already exists with the same kind: from %q to %q", edge.From().Metadata.Name, edge.To().Metadata.Name)
	}

	outgoing[e.To().Metadata.Name] = e

	return nil
}

// EdgesFrom returns the set of edges as a map of "from" phase names to
// map of "to" phase names to the edge instance itself.
func (p *Pipeline) EdgesFrom() map[string]map[string]Edge {
	return p.edges
}

// Edges returns all edges as a sequence
func (p *Pipeline) Edges() iter.Seq[Edge] {

	return iter.Seq[Edge](func(yield func(Edge) bool) {
		for _, out := range p.edges {
			for _, edge := range out {
				if !yield(edge) {
					return
				}
			}
		}
	})
}

// HistoryOptions are options for filtering history entries.
type HistoryOptions struct {
	Start uuid.UUID
}

// WithStart returns a HistoryOption that filters history entries after the given
// version uuid.
func WithStart(u uuid.UUID) containers.Option[HistoryOptions] {
	return func(opts *HistoryOptions) {
		opts.Start = u
	}
}

// PhaseOptions scopes a call to get phases from a pipeline.
type PhaseOptions struct {
	phase  Phase
	name   string
	labels map[string]string
}

// Matches returns true if the provided Phase matches the phase options
// set of conditions.
// An empty set of conditions always returns true.
func (p *PhaseOptions) Matches(phase Phase) bool {
	if p.phase != nil && phase != p.phase {
		return false
	}

	if !hasAllLabels(phase, p.labels) {
		return false
	}

	if p.name != "" && p.name != phase.Descriptor().Metadata.Name {
		return false
	}

	return true
}

// IsPhase causes a call to Phases to list specifically the provided phase p.
func IsPhase(p Phase) containers.Option[PhaseOptions] {
	return func(co *PhaseOptions) {
		co.phase = p
	}
}

// HasName causes a call to Phases to list any phase with the matching name.
func HasName(name string) containers.Option[PhaseOptions] {
	return func(co *PhaseOptions) {
		co.name = name
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

// hasAllLabels returns true if the provided phase has all the supplied labels
func hasAllLabels(c Phase, toFind map[string]string) (found bool) {
	labels := c.Descriptor().Metadata.Labels
	for k, v := range toFind {
		if found, ok := labels[k]; !ok || v != found {
			return false
		}
	}
	return true
}
