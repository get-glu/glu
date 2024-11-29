package reflog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/get-glu/glu/pkg/core"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

const (
	versionBucket = "v0"
	blobsBucket   = "blobs"
	refBucket     = "refs"
)

var (
	ErrBucketNotFound = errors.New("bucket not found")
)

type Log[R core.Resource] struct {
	db      *bbolt.DB
	encoder func(any) ([]byte, error)
	decoder func([]byte, any) error
	last    map[string]map[string]version
}

func New[R core.Resource](db *bbolt.DB) *Log[R] {
	return &Log[R]{
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

func (l *Log[R]) CreateReference(ctx context.Context, pipeline, phase core.Metadata) error {
	return l.db.Update(func(tx *bbolt.Tx) error {
		if _, err := createBucketPath(tx, versionBucket, refBucket, pipeline.Name, phase.Name); err != nil {
			return err
		}

		_, err := createBucketPath(tx, versionBucket, blobsBucket, pipeline.Name, phase.Name)
		return err
	})
}

func (l *Log[R]) RecordLatest(ctx context.Context, pipeline, phase core.Metadata, resource R, annotations map[string]string) error {
	slog := slog.With("pipeline", pipeline.Name, "phase", phase.Name)
	return l.db.Update(func(tx *bbolt.Tx) error {
		digest, err := resource.Digest()
		if err != nil {
			return err
		}

		refs, err := getRefsBucket(pipeline, phase, tx)
		if err != nil {
			return err
		}

		curLatest, ok := l.getLatestVersion(refs, pipeline, phase)
		if ok && bytes.Equal(curLatest.Digest, []byte(digest)) {
			slog.Debug("skipped recording latest", "reason", "NoChange")
			return nil
		}

		blobs, err := getBlobBucket(pipeline, phase, tx)
		if err != nil {
			return err
		}

		// insert encoded resource if digest not already persisted
		if v := blobs.Get([]byte(digest)); v == nil {
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

func (l *Log[R]) getLatestVersion(refs *bbolt.Bucket, pipeline, phase core.Metadata) (version, bool) {
	phases, ok := l.last[pipeline.Name]
	if !ok {
		phases = map[string]version{}
		l.last[phase.Name] = phases
	}

	version, ok := phases[phase.Name]
	if !ok {
		version, ok = l.fetchLatestVersion(refs)
		if !ok {
			return version, false
		}

		phases[phase.Name] = version
	}

	return version, true
}

func (l *Log[R]) fetchLatestVersion(refs *bbolt.Bucket) (v version, _ bool) {
	k, data := refs.Cursor().Last()
	if k == nil {
		return v, false
	}

	if err := l.decoder(data, &v); err != nil {
		return v, false
	}

	return v, true
}

func (l *Log[R]) History(ctx context.Context, pipeline, phase core.Metadata) (states []core.State, _ error) {
	return states, l.db.View(func(tx *bbolt.Tx) error {
		refs, err := getRefsBucket(pipeline, phase, tx)
		if err != nil {
			return err
		}

		blobs, err := getBlobBucket(pipeline, phase, tx)
		if err != nil {
			return err
		}

		// run a cursor in reverse to descend from most recent (largest) to oldest (smallest)
		cursor := refs.Cursor()
		for k, v := cursor.Last(); k != nil; k, v = cursor.Prev() {
			id, err := uuid.ParseBytes(k)
			if err != nil {
				return err
			}

			var version version
			if err := l.decoder(v, &version); err != nil {
				return err
			}

			var r R
			if blob := blobs.Get(version.Digest); blob != nil {
				if err := l.decoder(blob, &r); err != nil {
					return err
				}
			}

			states = append(states, core.State{
				Version:     id,
				Resource:    r,
				Annotations: version.Annotations,
			})
		}

		return nil
	})
}

func getBlobBucket(pipeline, phase core.Metadata, tx *bbolt.Tx) (*bbolt.Bucket, error) {
	return getBucket(tx, versionBucket, blobsBucket, pipeline.Name, phase.Name)
}

func getRefsBucket(pipeline, phase core.Metadata, tx *bbolt.Tx) (*bbolt.Bucket, error) {
	return getBucket(tx, versionBucket, refBucket, pipeline.Name, phase.Name)
}

func getBucket(tx *bbolt.Tx, path ...string) (bkt *bbolt.Bucket, err error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty path: %w", ErrBucketNotFound)
	}

	var b interface {
		Bucket([]byte) *bbolt.Bucket
	} = tx

	for i, p := range path {
		if bkt = b.Bucket([]byte(p)); bkt == nil {
			return nil, fmt.Errorf("bucket %q: %w", strings.Join(path[:i+1], "/"), ErrBucketNotFound)
		}
		b = bkt
	}
	return
}

func createBucketPath(tx *bbolt.Tx, path ...string) (bkt *bbolt.Bucket, err error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty path: %w", ErrBucketNotFound)
	}

	var b interface {
		CreateBucketIfNotExists([]byte) (*bbolt.Bucket, error)
	} = tx

	for i, p := range path {
		if bkt, err = b.CreateBucketIfNotExists([]byte(p)); err != nil {
			return nil, fmt.Errorf("bucket %q: %w", strings.Join(path[:i+1], "/"), err)
		}

		b = bkt
	}
	return
}
