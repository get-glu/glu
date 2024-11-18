package oci

import (
	"context"

	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/phases"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

var _ phases.Source[Resource] = (*Source[Resource])(nil)

type Resource interface {
	core.Resource
	ReadFromOCIDescriptor(v1.Descriptor) error
}

type Resolver interface {
	Resolve(_ context.Context) (v1.Descriptor, error)
}

type Source[R Resource] struct {
	resolver Resolver
}

func New[R Resource](resolver Resolver) *Source[R] {
	return &Source[R]{
		resolver: resolver,
	}
}

func (s *Source[R]) Type() string {
	return "oci"
}

func (s *Source[R]) View(ctx context.Context, _, _ core.Metadata, r R) error {
	desc, err := s.resolver.Resolve(ctx)
	if err != nil {
		return err
	}

	return r.ReadFromOCIDescriptor(desc)
}
