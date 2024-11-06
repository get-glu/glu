package oci

import (
	"context"
	"log/slog"

	"github.com/get-glu/glu/pkg/core"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type Derivable interface {
	ReadFromOCIDescriptor(v1.Descriptor) error
}

type Resolver interface {
	Resolve(_ context.Context) (v1.Descriptor, error)
}

type Source[A any, P interface {
	*A
	Derivable
}] struct {
	resolver Resolver
	meta     core.Metadata
	fn       func(core.Metadata) P
	last     P
}

type Registry interface {
	Register(core.Reconciler)
}

func New[A any, P interface {
	*A
	Derivable
}](
	pipeline Registry,
	resolver Resolver,
	meta core.Metadata,
	fn func(core.Metadata) P,
) (*Source[A, P], error) {
	src := &Source[A, P]{
		resolver: resolver,
		meta:     meta,
		fn:       fn,
	}

	pipeline.Register(src)

	return src, nil
}

func (s *Source[A, P]) Metadata() core.Metadata {
	return s.meta
}

func (s *Source[A, P]) Get(ctx context.Context) (any, error) {
	return s.GetResource(ctx)
}

func (s *Source[A, P]) GetResource(ctx context.Context) (P, error) {
	if s.last == nil {
		if err := s.Reconcile(ctx); err != nil {
			return nil, err
		}
	}

	return s.last, nil
}

func (s *Source[A, P]) Reconcile(ctx context.Context) error {
	slog.Debug("Reconcile", "type", "oci", "name", s.meta.Name)

	desc, err := s.resolver.Resolve(ctx)
	if err != nil {
		return err
	}

	p := s.fn(s.meta)
	if err := p.ReadFromOCIDescriptor(desc); err != nil {
		return err
	}

	s.last = p

	return nil

}

func getRepository(repo string) (*remote.Repository, error) {
	remote, err := remote.NewRepository(repo)
	if err != nil {
		return nil, err
	}

	creds, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err != nil {
		return nil, err
	}

	remote.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(creds),
	}

	return remote, nil
}
