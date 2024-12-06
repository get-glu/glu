package oci

import (
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	_ ResourceFromIndex    = (*BaseResource)(nil)
	_ ResourceFromManifest = (*BaseResource)(nil)
)

type BaseResource struct {
	// ImageName   string // TODO: add this when we have a use case for it
	ImageDigest digest.Digest `json:"image_digest,omitempty"`
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
