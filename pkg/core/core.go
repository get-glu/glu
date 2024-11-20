package core

import (
	"context"
	"errors"
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
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Pipeline is a collection of phases with potential promotion dependencies
// relationships between one another.
type Pipeline interface {
	Metadata() Metadata
	PhaseByName(string) (Phase, error)
	Phases(...containers.Option[PhaseOptions]) iter.Seq[Phase]
	Dependencies() map[Phase]Phase
}

// Resource is an instance of a resource in a phase.
// Primarilly, it exposes a Digest method used to produce
// a hash digest of the resource instances current state.
type Resource interface {
	Digest() (string, error)
}

// Phase is the core interface for resource sourcing and management.
// These types can be registered on pipelines and can depend upon on another for promotion.
type Phase interface {
	Metadata() Metadata
	Source() Metadata
	Get(context.Context) (Resource, error)
	Promote(context.Context) error
}

// AddPhaseOptions are used to configure the addition of a ResourcePhase to a Pipeline
type AddPhaseOptions[R Resource] struct {
	PromotedFrom ResourcePhase[R]
}

// PromotesFrom configures a dependent Phase to promote from for the Phase being added.
func PromotesFrom[R Resource](c ResourcePhase[R]) containers.Option[AddPhaseOptions[R]] {
	return func(o *AddPhaseOptions[R]) {
		o.PromotedFrom = c
	}
}

// ResourcePhase is a Phase bound to a particular resource type R.
type ResourcePhase[R Resource] interface {
	Phase
	GetResource(context.Context) (R, error)
}

// PhaseOptions scopes a call to get phases from a pipeline.
type PhaseOptions struct {
	phase  Phase
	labels map[string]string
}

func (p *PhaseOptions) Matches(phase Phase) bool {
	if p.phase != nil && phase != p.phase {
		return false
	}

	if !hasAllLabels(phase, p.labels) {
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
	labels := c.Metadata().Labels
	for k, v := range toFind {
		if found, ok := labels[k]; !ok || v != found {
			return false
		}
	}
	return true
}
