package oci

import (
	"context"
	"log/slog"

	"github.com/flipt-io/glu"
	"github.com/flipt-io/glu/pkg/core"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type Derivable interface {
	ReadFromOCIDescriptor(v1.Descriptor) error
}

type Source[A any, P interface {
	*A
	Derivable
}] struct {
	remote *remote.Repository
	meta   glu.Metadata
	fn     func(glu.Metadata) P
	last   P
}

type Registry interface {
	Register(core.Reconciler)
}

func New[A any, P interface {
	*A
	Derivable
}](
	pipeline Registry,
	image string,
	meta glu.Metadata,
	fn func(glu.Metadata) P,
) (*Source[A, P], error) {
	r, err := getRepository(image)
	if err != nil {
		return nil, err
	}

	src := &Source[A, P]{
		remote: r,
		meta:   meta,
		fn:     fn,
	}

	pipeline.Register(src)

	return src, nil
}

func (s *Source[A, P]) Metadata() glu.Metadata {
	return s.meta
}

func (s *Source[A, P]) GetAny(ctx context.Context) (any, error) {
	return s.Get(ctx)
}

func (s *Source[A, P]) Get(ctx context.Context) (P, error) {
	if s.last == nil {
		if err := s.Reconcile(ctx); err != nil {
			return nil, err
		}
	}

	return s.last, nil
}

func (s *Source[A, P]) Reconcile(ctx context.Context) error {
	slog.Debug("Reconcile", "type", "oci", "name", s.meta.Name)

	desc, err := s.remote.Resolve(ctx, s.remote.Reference.Reference)
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
