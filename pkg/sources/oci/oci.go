package oci

import (
	"context"
	"log/slog"
	"time"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type Source[A any] struct {
	remote *remote.Repository
	fn     func(v1.Descriptor) (A, error)
}

func New[A any](repo string, fn func(v1.Descriptor) (A, error)) (*Source[A], error) {
	r, err := getRepository(repo)
	if err != nil {
		return nil, err
	}

	return &Source[A]{
		remote: r,
		fn:     fn,
	}, nil
}

func (s *Source[A]) Subscribe(ctx context.Context, ch chan<- A) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ticker.C:
				desc, err := s.remote.Resolve(ctx, s.remote.Reference.Reference)
				if err != nil {
					slog.Error("fetching descriptor for remote", "error", err)
					continue
				}

				a, err := s.fn(desc)
				if err != nil {
					slog.Error("transforming descriptor to target", "error", err)
					continue
				}

				ch <- a
			case <-ctx.Done():
				return
			}
		}
	}()
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
