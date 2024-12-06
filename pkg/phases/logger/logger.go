package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/core/typed"
	"github.com/get-glu/glu/pkg/kv"
	"github.com/google/uuid"
)

const (
	versionBucket = "v0"
	blobsBucket   = "blobs"
	refBucket     = "refs"
)

var (
	ErrNotFound = errors.New("not found")

	_ typed.PhaseLogger[core.Resource] = (*PhaseLogger[core.Resource])(nil)
)

type PhaseLogger[R core.Resource] struct {
	db      kv.DB
	encoder func(any) ([]byte, error)
	decoder func([]byte, any) error
	last    map[string]map[string]version
}

func New[R core.Resource](db kv.DB) *PhaseLogger[R] {
	return &PhaseLogger[R]{
		db:      db,
		encoder: json.Marshal,
		decoder: json.Unmarshal,
		last:    map[string]map[string]version{},
	}
}

type version struct {
	Digest      []byte
	Annotations map[string]string
}

func (l *PhaseLogger[R]) CreateLog(ctx context.Context, phase core.Descriptor) error {
	return l.db.Update(func(tx kv.Tx) error {
		if _, err := createBucketPath(tx, versionBucket, refBucket, phase.Pipeline, phase.Metadata.Name); err != nil {
			return err
		}

		_, err := createBucketPath(tx, versionBucket, blobsBucket, phase.Pipeline, phase.Metadata.Name)
		return err
	})
}

func (l *PhaseLogger[R]) RecordLatest(ctx context.Context, phase core.Descriptor, resource R, annotations map[string]string) error {
	digest, err := resource.Digest()
	if err != nil {
		return err
	}

	// check if we can skip the write if we're already up to date
	var upToDate bool
	if err := l.db.View(func(tx kv.Tx) error {
		refs, err := getRefsBucket(phase, tx)
		if err != nil {
			return err
		}

		upToDate = l.isUpToDate(refs, phase, digest)

		return nil
	}); err != nil || upToDate {
		return err
	}

	return l.db.Update(func(tx kv.Tx) error {
		refs, err := getRefsBucket(phase, tx)
		if err != nil {
			return err
		}

		// check again now that we have a write lock if we can skip the update
		if l.isUpToDate(refs, phase, digest) {
			return nil
		}

		blobs, err := getBlobBucket(phase, tx)
		if err != nil {
			return err
		}

		// insert encoded resource if digest not already persisted
		if _, err := blobs.Get([]byte(digest)); errors.Is(err, kv.ErrNotFound) {
			data, err := l.encoder(resource)
			if err != nil {
				return err
			}

			if err := blobs.Put([]byte(digest), data); err != nil {
				return err
			}
		}

		encoded, err := l.encoder(version{[]byte(digest), annotations})
		if err != nil {
			return err
		}

		id, err := uuid.NewV7()
		if err != nil {
			return err
		}

		idBytes, err := id.MarshalText()
		if err != nil {
			return err
		}

		return refs.Put(idBytes, encoded)
	})
}

func (l *PhaseLogger[R]) isUpToDate(refs kv.Bucket, phase core.Descriptor, digest string) bool {
	slog := slog.With("pipeline", phase.Pipeline, "phase", phase.Metadata.Name)

	curLatest, ok := l.getLatestVersion(refs, phase)
	if ok && bytes.Equal(curLatest.Digest, []byte(digest)) {
		slog.Debug("skipped recording latest", "reason", "NoChange")
		return true
	}

	return false
}

// GetLatestResource returns the state of the latest resource recorded.
func (l *PhaseLogger[R]) GetLatestResource(ctx context.Context, phase core.Descriptor) (r R, _ error) {
	return r, l.db.View(func(tx kv.Tx) error {
		refs, err := getRefsBucket(phase, tx)
		if err != nil {
			return err
		}

		curLatest, ok := l.getLatestVersion(refs, phase)
		if !ok {
			return fmt.Errorf("latest version: %w", ErrNotFound)
		}

		blobs, err := getBlobBucket(phase, tx)
		if err != nil {
			return err
		}

		blob, err := blobs.Get(curLatest.Digest)
		if err != nil {
			return fmt.Errorf("version data for %q: %w", curLatest.Digest, err)
		}

		return l.decoder(blob, &r)
	})
}

// GetResourceAtVersion returns the state of the resource at a given point in history identified by the provided version
func (l *PhaseLogger[R]) GetResourceAtVersion(ctx context.Context, phase core.Descriptor, v uuid.UUID) (r R, _ error) {
	return r, l.db.View(func(tx kv.Tx) error {
		refs, err := getRefsBucket(phase, tx)
		if err != nil {
			return err
		}

		versionBytes, err := v.MarshalText()
		if err != nil {
			return err
		}

		versionData, err := refs.Get(versionBytes)
		if err != nil {
			return fmt.Errorf("version %q: %w", v, err)
		}

		blobs, err := getBlobBucket(phase, tx)
		if err != nil {
			return err
		}

		var version version
		if err := l.decoder(versionData, &version); err != nil {
			return err
		}

		blob, err := blobs.Get(version.Digest)
		if err != nil {
			return fmt.Errorf("version data for %q: %w", v, err)
		}

		return l.decoder(blob, &r)
	})
}

func (l *PhaseLogger[R]) getLatestVersion(refs kv.Bucket, phase core.Descriptor) (version, bool) {
	phases, ok := l.last[phase.Pipeline]
	if !ok {
		phases = map[string]version{}
		l.last[phase.Metadata.Name] = phases
	}

	version, ok := phases[phase.Metadata.Name]
	if !ok {
		version, ok = l.fetchLatestVersion(refs)
		if !ok {
			return version, false
		}

		phases[phase.Metadata.Name] = version
	}

	return version, true
}

func (l *PhaseLogger[R]) fetchLatestVersion(refs kv.Bucket) (v version, _ bool) {
	_, data, err := refs.Last()
	if err != nil {
		return v, false
	}

	if err := l.decoder(data, &v); err != nil {
		return v, false
	}

	return v, true
}

// Histort returns a slice of states for a provided phase descriptor.
func (l *PhaseLogger[R]) History(ctx context.Context, phase core.Descriptor) (states []core.State, _ error) {
	return states, l.db.View(func(tx kv.Tx) error {
		refs, err := getRefsBucket(phase, tx)
		if err != nil {
			return err
		}

		blobs, err := getBlobBucket(phase, tx)
		if err != nil {
			return err
		}

		for k, v := range refs.Range(kv.WithOrder(kv.Descending)) {
			id, err := uuid.ParseBytes(k)
			if err != nil {
				return err
			}

			var version version
			if err := l.decoder(v, &version); err != nil {
				return err
			}

			blob, err := blobs.Get(version.Digest)
			if err != nil {
				return err
			}

			var r R
			if err := l.decoder(blob, &r); err != nil {
				return err
			}

			timestamp := time.Unix(id.Time().UnixTime())
			states = append(states, core.State{
				Version:     id,
				Digest:      string(version.Digest),
				Resource:    r,
				Annotations: version.Annotations,
				RecordedAt:  timestamp.UTC(),
			})
		}

		return nil
	})
}

func getBlobBucket(phase core.Descriptor, tx kv.Tx) (kv.Bucket, error) {
	return getBucket(tx, versionBucket, blobsBucket, phase.Pipeline, phase.Metadata.Name)
}

func getRefsBucket(phase core.Descriptor, tx kv.Tx) (kv.Bucket, error) {
	return getBucket(tx, versionBucket, refBucket, phase.Pipeline, phase.Metadata.Name)
}

func getBucket(tx kv.Tx, path ...string) (bkt kv.Bucket, err error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty path: %w", ErrNotFound)
	}

	var b interface {
		Bucket([]byte) (kv.Bucket, error)
	} = tx

	for _, p := range path {
		if bkt, err = b.Bucket([]byte(p)); err != nil {
			return nil, err
		}
		b = bkt
	}
	return
}

func createBucketPath(tx kv.Tx, path ...string) (bkt kv.Bucket, err error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty path: %w", ErrNotFound)
	}

	var b interface {
		CreateBucketIfNotExists([]byte) (kv.Bucket, error)
	} = tx

	for _, p := range path {
		if bkt, err = b.CreateBucketIfNotExists([]byte(p)); err != nil {
			return nil, err
		}

		b = bkt
	}
	return
}
