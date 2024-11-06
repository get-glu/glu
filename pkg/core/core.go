package core

import (
	"context"
)

// Controller is the core interface for a resource reconciler.
// These types can be registered on pipelines and dependend upon on another.
type Controller interface {
	Metadata() Metadata
	Get(context.Context) (any, error)
	Reconcile(context.Context) error
}

// Resource is an instance of a resource in a phase
// It exposes its metadata, unique current digest and functionality
// for extracting from updating a filesystem with the current version.
type Resource interface {
	Metadata() *Metadata
	Digest() (string, error)
}

// Pipeline is a set of reconcilers organised into phases.
type Pipeline struct {
	ctx  context.Context
	name string

	reconcilers []Controller
}

// NewPipeline constructs a new, empty named pipeline .
func NewPipeline(ctx context.Context, name string) *Pipeline {
	return &Pipeline{
		ctx:  ctx,
		name: name,
	}
}

// Name returns the name of the pipeline as a string.
func (p *Pipeline) Name() string {
	return p.name
}

// Register adds a reconciler to the pipeline.
func (p *Pipeline) Register(r Controller) {
	p.reconcilers = append(p.reconcilers, r)
}

// Phases returns all reconcilers as a map indexed by phase
// and then reconciler resource name.
func (p *Pipeline) Phases() map[string]map[string]Controller {
	phases := map[string]map[string]Controller{}
	for _, r := range p.reconcilers {
		meta := r.Metadata()
		phase, ok := phases[meta.Phase]
		if !ok {
			phase = map[string]Controller{}
		}

		phase[meta.Name] = r

		phases[meta.Phase] = phase
	}
	return phases
}

// Metadata contains the unique information used to identify
// a named resource instance in a particular phase.
type Metadata struct {
	Name   string
	Phase  string
	Labels map[string]string
}
