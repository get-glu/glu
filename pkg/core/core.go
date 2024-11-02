package core

import (
	"context"

	"github.com/flipt-io/glu/pkg/fs"
)

type Reconciler interface {
	Metadata() Metadata
	GetAny(context.Context) (any, error)
	Reconcile(context.Context) error
}

type Repository interface {
	View(context.Context, Resource) error
	Update(_ context.Context, from, to Resource) error
}

type Pipeline struct {
	ctx  context.Context
	name string

	reconcilers []Reconciler
}

func NewPipeline(ctx context.Context, name string) *Pipeline {
	return &Pipeline{
		ctx:  ctx,
		name: name,
	}
}

func (p *Pipeline) Name() string {
	return p.name
}

func (p *Pipeline) Register(r Reconciler) {
	p.reconcilers = append(p.reconcilers, r)
}

func (p *Pipeline) Phases() map[string]map[string]Reconciler {
	phases := map[string]map[string]Reconciler{}
	for _, r := range p.reconcilers {
		meta := r.Metadata()
		phase, ok := phases[meta.Phase]
		if !ok {
			phase = map[string]Reconciler{}
		}

		phase[meta.Name] = r

		phases[meta.Phase] = phase
	}
	return phases
}

type Metadata struct {
	Name   string
	Phase  string
	Labels map[string]string
}

type Resource interface {
	Metadata() *Metadata
	Digest() (string, error)
	ReadFrom(context.Context, fs.Filesystem) error
	WriteTo(context.Context, fs.Filesystem) error
}

type Proposal struct {
	BaseRevision string
	BaseBranch   string
	Branch       string
	Digest       string
	Title        string
	Body         string

	ExternalMetadata map[string]any
}
