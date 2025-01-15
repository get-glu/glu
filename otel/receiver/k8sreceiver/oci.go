package k8sreceiver

import (
	"context"
	"encoding/json"
	"io"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"
)

func fetchOCIAttributes(ctx context.Context, ref string) (map[string]string, error) {
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, err
	}

	desc, reader, err := repo.FetchReference(ctx, ref)
	if err != nil {
		return nil, err
	}

	defer func() {
		// discard reader contents and close before returning
		_, _ = io.Copy(io.Discard, reader)
		reader.Close()
	}()

	switch desc.MediaType {
	case v1.MediaTypeImageIndex:
		var index v1.Index
		if err := json.NewDecoder(reader).Decode(&index); err != nil {
			return nil, err
		}

		return index.Annotations, nil
	case v1.MediaTypeImageManifest:
		var manifest v1.Manifest
		if err := json.NewDecoder(reader).Decode(&manifest); err != nil {
			return nil, err
		}

		return manifest.Annotations, nil
	default:
	}

	rest, err := content.ReadAll(reader, desc)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rest, &desc); err != nil {
		return nil, err
	}

	return desc.Annotations, nil
}
