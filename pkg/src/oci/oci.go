package oci

import (
	"context"
	"encoding/json"
	"io"

	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/phases"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
)

const ANNOTATION_OCI_IMAGE_URL = "dev.getglu.oci.image.url"

var _ phases.Source[Resource] = (*Source[Resource])(nil)

type Resource interface {
	core.Resource
	ReadFromOCIDescriptor(v1.Descriptor, io.ReadCloser) error
}

type Resolver interface {
	Resolve(_ context.Context) (v1.Descriptor, io.ReadCloser, error)
	Reference() string
}

type Source[R Resource] struct {
	resolver Resolver
}

func New[R Resource](resolver Resolver) *Source[R] {
	return &Source[R]{
		resolver: resolver,
	}
}

func (s *Source[R]) Metadata() core.Metadata {
	return core.Metadata{
		Name:        "oci",
		Annotations: map[string]string{ANNOTATION_OCI_IMAGE_URL: s.resolver.Reference()},
	}
}

func (s *Source[R]) View(ctx context.Context, _, _ core.Metadata, r R) error {
	desc, reader, err := s.resolver.Resolve(ctx)
	if err != nil {
		return err
	}

	return r.ReadFromOCIDescriptor(desc, reader)
}

type BaseResource struct {
	// ImageName   string // TODO: add this when we have a use case for it
	ImageDigest digest.Digest
	annotations map[string]string
}

func (r *BaseResource) Digest() (string, error) {
	return r.ImageDigest.Encoded(), nil
}

func (r *BaseResource) Annotations() map[string]string {
	return r.annotations
}

func (r *BaseResource) ReadFromOCIDescriptor(desc v1.Descriptor, reader io.ReadCloser) error {
	r.ImageDigest = desc.Digest

	defer reader.Close()

	b, err := content.ReadAll(reader, desc)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, &desc); err != nil {
		return err
	}

	r.annotations = desc.Annotations
	return nil
}
