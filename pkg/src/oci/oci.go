package oci

import (
	"context"

	"github.com/get-glu/glu/internal/oci"
	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/controllers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/credentials"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

var _ controllers.Source[Resource] = (*Source[Resource])(nil)

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

type ConfigSource interface {
	OCIRepositoryConfig(name string) (*config.OCIRepository, error)
	GetCredential(name string) (*credentials.Credential, error)
}

func New[R Resource](name string, cconf ConfigSource) (*Source[R], error) {
	conf, err := cconf.OCIRepositoryConfig(name)
	if err != nil {
		return nil, err
	}

	var cred *credentials.Credential
	if conf.Credential != "" {
		cred, err = cconf.GetCredential(conf.Credential)
		if err != nil {
			return nil, err
		}
	}

	resolver, err := oci.New(conf.Reference, cred)
	if err != nil {
		return nil, err
	}

	src := &Source[R]{
		resolver: resolver,
	}

	return src, nil
}

func (s *Source[R]) View(ctx context.Context, _ core.Metadata, r R) error {
	desc, err := s.resolver.Resolve(ctx)
	if err != nil {
		return err
	}

	return r.ReadFromOCIDescriptor(desc)
}
