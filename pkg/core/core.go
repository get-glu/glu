package core

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when a particular resource cannot be located
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned when an attempt is made to create a resource
	// which already exists
	ErrAlreadyExists = errors.New("already exists")
	// ErrNoChange is returned when an update produced zero changes
	ErrNoChange = errors.New("update produced no change")
)

// Metadata contains the unique information used to identify
// a named resource instance in a particular phase.
type Metadata struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Resource is an instance of a resource in a phase.
// Primarilly, it exposes a Digest method used to produce
// a hash digest of the resource instances current state.
type Resource interface {
	Digest() (string, error)
}

// ResourceWithAnnotations is a resource with additional annotations
type ResourceWithAnnotations interface {
	Resource
	Annotations() map[string]string
}

// Descriptor is a type which describes a Phase
type Descriptor struct {
	Kind     string   `json:"kind"`
	Pipeline string   `json:"pipeline"`
	Metadata Metadata `json:"metadata"`
}

func (d Descriptor) String() string {
	return fmt.Sprintf("%s/%s", d.Pipeline, d.Metadata.Name)
}
