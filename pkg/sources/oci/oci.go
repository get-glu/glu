package oci

import (
	"context"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type OCIDerivable interface {
	ReadFromOCIDescriptor(v1.Descriptor) error
}

type Source[A any, P interface {
	*A
	OCIDerivable
}] struct {
	remote *remote.Repository
}

func New[A any, P interface {
	*A
	OCIDerivable
}](repo string, p P) (*Source[A, P], error) {
	r, err := getRepository(repo)
	if err != nil {
		return nil, err
	}

	return &Source[A, P]{
		remote: r,
	}, nil
}

func (s *Source[A, P]) Reconcile(ctx context.Context) (P, error) {
	desc, err := s.remote.Resolve(ctx, s.remote.Reference.Reference)
	if err != nil {
		return nil, err
	}

	p := P(new(A))
	if err := p.ReadFromOCIDescriptor(desc); err != nil {
		return nil, err
	}

	return p, nil

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
