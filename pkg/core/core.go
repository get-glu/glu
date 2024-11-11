package core

import (
	"context"
	"errors"

	"github.com/get-glu/glu/pkg/containers"
)

var ErrProposalNotFound = errors.New("proposal not found")

// Metadata contains the unique information used to identify
// a named resource instance in a particular phase.
type Metadata struct {
	Name   string
	Labels map[string]string
}

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
	Digest() (string, error)
}

type Pipeline[R Resource] struct {
	meta         Metadata
	newFn        func() R
	dependencies map[ResourceController[R]]AddOptions[R]
}

func NewPipeline[R Resource](meta Metadata, newFn func() R) *Pipeline[R] {
	return &Pipeline[R]{
		meta:         meta,
		newFn:        newFn,
		dependencies: map[ResourceController[R]]AddOptions[R]{},
	}
}

type AddOptions[R Resource] struct {
	dependsOn ResourceController[R]
}

func PromotesFrom[R Resource](c ResourceController[R]) containers.Option[AddOptions[R]] {
	return func(ao *AddOptions[R]) {
		ao.dependsOn = c
	}
}

type ResourceController[R Resource] interface {
	Controller
	GetResource(context.Context) (R, error)
}

func (p *Pipeline[R]) New() R {
	return p.newFn()
}

func (p *Pipeline[R]) Metadata() Metadata {
	return p.meta
}

func (p *Pipeline[R]) Add(r ResourceController[R], opts ...containers.Option[AddOptions[R]]) {
	var add AddOptions[R]
	containers.ApplyAll(&add, opts...)

	p.dependencies[r] = add
}

func (p *Pipeline[R]) GetDependency(r ResourceController[R]) (ResourceController[R], bool) {
	opts, ok := p.dependencies[r]
	if ok && opts.dependsOn != nil {
		return opts.dependsOn, true
	}

	return nil, false
}

func (p *Pipeline[R]) Controllers() map[string]Controller {
	controllers := map[string]Controller{}
	for k := range p.dependencies {
		controllers[k.Metadata().Name] = k
	}

	return controllers
}

func (p *Pipeline[R]) Dependencies() map[Controller]Controller {
	deps := map[Controller]Controller{}
	for k, v := range p.dependencies {
		deps[k] = v.dependsOn
	}

	return deps
}
