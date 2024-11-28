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
	ReadFromOCIDescriptor(v1.Descriptor) error
}

type ResourceFromManifest interface {
	Resource
	ReadFromOCIManifest(v1.Descriptor, v1.Manifest) error
}

type ResourceFromIndex interface {
	Resource
	ReadFromOCIIndex(v1.Descriptor, v1.Index) error
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

	defer func() {
		// discard reader contents and close before returning
		io.Copy(io.Discard, reader)
		reader.Close()
	}()

	switch desc.MediaType {
	case v1.MediaTypeImageIndex:
		ri, ok := Resource(r).(ResourceFromIndex)
		if !ok {
			break
		}

		var index v1.Index
		if err := json.NewDecoder(reader).Decode(&index); err != nil {
			return err
		}

		return ri.ReadFromOCIIndex(desc, index)
	case v1.MediaTypeImageManifest:
		rm, ok := Resource(r).(ResourceFromManifest)
		if !ok {
			break
		}

		var manifest v1.Manifest
		if err := json.NewDecoder(reader).Decode(&manifest); err != nil {
			return err
		}

		return rm.ReadFromOCIManifest(desc, manifest)
	default:
	}

	rest, err := content.ReadAll(reader, desc)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(rest, &desc); err != nil {
		return err
	}

	return r.ReadFromOCIDescriptor(desc)
}

var (
	_ ResourceFromIndex    = (*BaseResource)(nil)
	_ ResourceFromManifest = (*BaseResource)(nil)
)

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

func (r *BaseResource) ReadFromOCIDescriptor(desc v1.Descriptor) error {
	r.ImageDigest = desc.Digest
	r.annotations = desc.Annotations
	return nil
}

func (r *BaseResource) ReadFromOCIManifest(desc v1.Descriptor, manifest v1.Manifest) error {
	r.ImageDigest = desc.Digest
	r.annotations = manifest.Annotations

	return nil
}

func (r *BaseResource) ReadFromOCIIndex(desc v1.Descriptor, index v1.Index) error {
	r.ImageDigest = desc.Digest
	r.annotations = index.Annotations
	return nil
}
