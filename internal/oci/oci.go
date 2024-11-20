package oci

import (
	"context"

	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/credentials"
	"github.com/get-glu/glu/pkg/src/oci"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
)

var _ oci.Resolver = (*Repository)(nil)

type Repository struct {
	repo *remote.Repository
	conf config.OCIRepository
}

func New(reference string, cred *credentials.Credential) (_ *Repository, err error) {
	repo, err := remote.NewRepository(reference)
	if err != nil {
		return nil, err
	}

	if cred != nil {
		repo.Client, err = cred.OCIClient(repo.Reference.Registry)
		if err != nil {
			return nil, err
		}
	}

	return &Repository{repo: repo}, nil
}

func (r *Repository) Resolve(ctx context.Context) (v1.Descriptor, error) {
	return r.repo.Resolve(ctx, r.repo.Reference.ReferenceOrDefault())
}

func (r *Repository) Reference() string {
	return r.repo.Reference.String()
}
