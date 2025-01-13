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
)

// Metadata contains the unique information used to identify
// a named resource instance in a particular phase.
type Metadata struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Name is a utility for quickly creating an instance of Metadata
// with a name (required).
func Name(name string) Metadata {
	return Metadata{Name: name, Labels: map[string]string{}, Annotations: map[string]string{}}
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
