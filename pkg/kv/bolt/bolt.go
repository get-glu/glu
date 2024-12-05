package bolt

import (
	"fmt"
	"io/fs"
	"iter"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/kv"
	"go.etcd.io/bbolt"
)

var _ kv.DB = (*DB)(nil)

type DB struct {
	db *bbolt.DB
}

func Open(path string, mode fs.FileMode, opts *bbolt.Options) (*DB, error) {
	db, err := bbolt.Open(path, mode, opts)
	if err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

func (d *DB) View(fn func(kv.Tx) error) error {
	return d.db.View(func(tx *bbolt.Tx) error {
		return fn(&Tx{tx: tx})
	})
}

func (d *DB) Update(fn func(kv.Tx) error) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		return fn(&Tx{tx: tx})
	})
}

type Tx struct {
	tx *bbolt.Tx
}

func (t *Tx) Bucket(name []byte) (kv.Bucket, error) {
	if bkt := t.tx.Bucket(name); bkt != nil {
		return &Bucket{bucket: bkt}, nil
	}

	return nil, fmt.Errorf("bucket %q: %w", string(name), kv.ErrNotFound)
}

func (t *Tx) CreateBucketIfNotExists(name []byte) (kv.Bucket, error) {
	bkt, err := t.tx.CreateBucketIfNotExists(name)
	if err != nil {
		return nil, err
	}

	return &Bucket{bucket: bkt}, nil
}

type Bucket struct {
	name   string
	bucket *bbolt.Bucket
}

func (b *Bucket) First() (k, v []byte, err error) {
	if k, v := b.bucket.Cursor().First(); k != nil {
		return k, v, nil
	}

	return nil, nil, fmt.Errorf("bucket %q first key: %w", b.name, kv.ErrNotFound)
}

func (b *Bucket) Last() (k, v []byte, err error) {
	if k, v := b.bucket.Cursor().Last(); k != nil {
		return k, v, nil
	}

	return nil, nil, fmt.Errorf("bucket %q last key: %w", b.name, kv.ErrNotFound)
}

func (b *Bucket) Bucket(name []byte) (kv.Bucket, error) {
	if bkt := b.bucket.Bucket(name); bkt != nil {
		return &Bucket{name: string(name), bucket: bkt}, nil
	}

	return nil, fmt.Errorf("bucket %q: %w", string(name), kv.ErrNotFound)
}

func (b *Bucket) CreateBucketIfNotExists(name []byte) (kv.Bucket, error) {
	bkt, err := b.bucket.CreateBucketIfNotExists(name)
	if err != nil {
		return nil, err
	}

	return &Bucket{bucket: bkt}, nil
}

func (b *Bucket) Get(key []byte) ([]byte, error) {
	if v := b.bucket.Get(key); v != nil {
		return v, nil
	}

	return nil, fmt.Errorf("bucket %q key %q: %w", b.name, string(key), kv.ErrNotFound)
}

func (b *Bucket) Put(k, v []byte) error {
	return b.bucket.Put(k, v)
}

func (b *Bucket) Range(opts ...containers.Option[kv.RangeOptions]) iter.Seq2[[]byte, []byte] {
	var options kv.RangeOptions
	containers.ApplyAll(&options, opts...)

	var (
		cursor = b.bucket.Cursor()
		k, v   []byte
	)

	switch options.Order {
	case kv.Descending:
		k, v = cursor.Last()
	default:
		// ascending
		k, v = cursor.First()
	}

	return iter.Seq2[[]byte, []byte](func(yield func(k, v []byte) bool) {
		if k == nil {
			return
		}

		if !yield(k, v) {
			return
		}

		switch options.Order {
		case kv.Descending:
			k, v = cursor.Prev()
		default:
			// ascending
			k, v = cursor.Next()
		}
	})
}
