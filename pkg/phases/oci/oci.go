package oci

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"time"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/core/typed"
	"github.com/get-glu/glu/pkg/kv/memory"
	"github.com/get-glu/glu/pkg/phases/logger"
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
	interval time.Duration
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
		logger:   logger.New[R](memory.New()),
		interval: 20 * time.Second,
	}

	containers.ApplyAll(phase, opts...)

	meta.Annotations[ANNOTATION_OCI_IMAGE_URL] = phase.resolver.Reference()

	if err := phase.logger.CreateLog(ctx, phase.Descriptor()); err != nil {
		return nil, err
	}

	// do initial fetch and set for resource
	if err := phase.updateResource(ctx); err != nil {
		return nil, err
	}

	ticker := time.NewTicker(phase.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := phase.updateResource(ctx); err != nil {
					slog.Error("updating phase resource", "type", "oci", "phase", meta.Name, "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return phase, nil
}

func (p *Phase[R]) Descriptor() core.Descriptor {
	return core.Descriptor{
		Kind:     "oci",
		Pipeline: p.pipeline,
		Metadata: p.meta,
	}
}

func (p *Phase[R]) Get(ctx context.Context) (core.Resource, error) {
	return p.GetResource(ctx)
}

func (p *Phase[R]) GetResource(ctx context.Context) (R, error) {
	return p.logger.GetLatestResource(ctx, p.Descriptor())
}

func (p *Phase[R]) updateResource(ctx context.Context) error {
	r, err := p.fetchResource(ctx)
	if err != nil {
		return err
	}

	return p.logger.RecordLatest(ctx, p.Descriptor(), r, nil)
}

func (p *Phase[R]) fetchResource(ctx context.Context) (R, error) {
	r := p.newFn()

	desc, reader, err := p.resolver.Resolve(ctx)
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

func (p *Phase[A]) History(ctx context.Context, opts ...containers.Option[core.HistoryOptions]) ([]core.State, error) {
	return p.logger.History(ctx, p.Descriptor(), opts...)
}
