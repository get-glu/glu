package core

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/get-glu/glu/pkg/containers"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

// Metadata contains the unique information used to identify
// a named resource instance in a particular phase.
type Metadata struct {
	Name   string
	Labels map[string]string
}

// Controller is the core interface for resource sourcing and management.
// These types can be registered on pipelines and can depend upon on another for promotion.
type Controller interface {
	Metadata() Metadata
	Get(context.Context) (any, error)
	Promote(context.Context) error
}

// Resource is an instance of a resource in a phase
// It exposes its metadata, unique current digest and functionality
// for extracting from updating a filesystem with the current version.
type Resource interface {
	Digest() (string, error)
}

type Pipeline[R Resource] struct {
	meta  Metadata
	newFn func() R
	nodes map[string]entry[R]
}

type entry[R Resource] struct {
	ResourceController[R]
	promotesFrom ResourceController[R]
}

func NewPipeline[R Resource](meta Metadata, newFn func() R) *Pipeline[R] {
	return &Pipeline[R]{
		meta:  meta,
		newFn: newFn,
		nodes: map[string]entry[R]{},
	}
}

type AddOptions[R Resource] struct {
	entry entry[R]
}

func PromotesFrom[R Resource](c ResourceController[R]) containers.Option[AddOptions[R]] {
	return func(ao *AddOptions[R]) {
		ao.entry.promotesFrom = c
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

func (p *Pipeline[R]) Add(r ResourceController[R], opts ...containers.Option[AddOptions[R]]) error {
	add := AddOptions[R]{entry: entry[R]{ResourceController: r}}
	containers.ApplyAll(&add, opts...)

	if _, existing := p.nodes[r.Metadata().Name]; existing {
		return fmt.Errorf("controller %q: %w", r.Metadata().Name, ErrAlreadyExists)
	}

	p.nodes[r.Metadata().Name] = add.entry

	return nil
}

func (p *Pipeline[R]) PromotedFrom(c ResourceController[R]) (ResourceController[R], bool) {
	dep, ok := p.nodes[c.Metadata().Name]
	if !ok || dep.promotesFrom == nil {
		return nil, false
	}

	return dep.promotesFrom, true
}

type ControllersOptions struct {
	controller Controller
	labels     map[string]string
}

func IsController(c Controller) containers.Option[ControllersOptions] {
	return func(co *ControllersOptions) {
		co.controller = c
	}
}

func HasLabel(k, v string) containers.Option[ControllersOptions] {
	return func(co *ControllersOptions) {
		if co.labels == nil {
			co.labels = map[string]string{}
		}

		co.labels[k] = v
	}
}

func HasAllLabels(labels map[string]string) containers.Option[ControllersOptions] {
	return func(co *ControllersOptions) {
		co.labels = labels
	}
}

func (p *Pipeline[R]) ControllerByName(name string) (Controller, error) {
	entry, ok := p.nodes[name]
	if !ok {
		return nil, fmt.Errorf("controller %q: %w", name, ErrNotFound)
	}

	return entry.ResourceController, nil
}

func (p *Pipeline[R]) Controllers(opts ...containers.Option[ControllersOptions]) iter.Seq[Controller] {
	var options ControllersOptions
	containers.ApplyAll(&options, opts...)

	return iter.Seq[Controller](func(yield func(Controller) bool) {
		for _, entry := range p.nodes {
			if options.controller != nil && entry.ResourceController != options.controller {
				continue
			}

			if !hasAllLabels(entry.ResourceController, options.labels) {
				continue
			}

			if !yield(entry.ResourceController) {
				break
			}
		}
	})
}

func (p *Pipeline[R]) Dependencies() map[Controller]Controller {
	deps := map[Controller]Controller{}
	for _, entry := range p.nodes {
		deps[entry.ResourceController] = entry.promotesFrom
	}

	return deps
}

// hasAllLabels returns true if the provided controller has all the supplied labels
func hasAllLabels(c Controller, toFind map[string]string) bool {
	labels := c.Metadata().Labels
	for k, v := range toFind {
		if found, ok := labels[k]; !ok || v != found {
			return false
		}
	}
	return true
}
