package oci

import (
	"context"
	"encoding/json"
	"io"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/core/typed"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
)

const ANNOTATION_OCI_IMAGE_URL = "dev.getglu.oci.image.url"

var _ typed.Phase[Resource] = (*Phase[Resource])(nil)

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

type Phase[R Resource] struct {
	pipeline string
	meta     core.Metadata
	newFn    func() R
	resolver Resolver
	logger   typed.PhaseLogger[R]
}

func New[R Resource](
	ctx context.Context,
	pipeline string,
	meta core.Metadata,
	newFn func() R,
	resolver Resolver,
	opts ...containers.Option[Phase[R]],
) (*Phase[R], error) {
	if meta.Annotations == nil {
		meta.Annotations = map[string]string{}
	}

	phase := &Phase[R]{
		pipeline: pipeline,
		meta:     meta,
		newFn:    newFn,
		resolver: resolver,
	}

	containers.ApplyAll(phase, opts...)

	meta.Annotations[ANNOTATION_OCI_IMAGE_URL] = phase.resolver.Reference()

	if phase.logger != nil {
		if err := phase.logger.CreateLog(ctx, phase.Descriptor()); err != nil {
			return nil, err
		}
	}

	return phase, nil
}

func (s *Phase[R]) Descriptor() core.Descriptor {
	return core.Descriptor{
		Kind:     "oci",
		Pipeline: s.pipeline,
		Metadata: s.meta,
	}
}

func (s *Phase[R]) Get(ctx context.Context) (core.Resource, error) {
	return s.GetResource(ctx)
}

func (s *Phase[R]) GetResource(ctx context.Context) (R, error) {
	r := s.newFn()

	desc, reader, err := s.resolver.Resolve(ctx)
	if err != nil {
		return r, err
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
			return r, err
		}

		return r, ri.ReadFromOCIIndex(desc, index)
	case v1.MediaTypeImageManifest:
		rm, ok := Resource(r).(ResourceFromManifest)
		if !ok {
			break
		}

		var manifest v1.Manifest
		if err := json.NewDecoder(reader).Decode(&manifest); err != nil {
			return r, err
		}

		return r, rm.ReadFromOCIManifest(desc, manifest)
	default:
	}

	rest, err := content.ReadAll(reader, desc)
	if err != nil {
		return r, err
	}

	if err := json.Unmarshal(rest, &desc); err != nil {
		return r, err
	}

	return r, r.ReadFromOCIDescriptor(desc)
}

func (p *Phase[A]) History(ctx context.Context) ([]core.State, error) {
	return p.logger.History(ctx, p.Descriptor())
}
