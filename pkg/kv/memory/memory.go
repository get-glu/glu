package memory

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"sync"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/kv"
	"github.com/google/btree"
)

type DB struct {
	mu   sync.RWMutex
	root *Bucket
}

func New() *DB {
	return &DB{root: newBucket("")}
}

func (d *DB) View(fn func(kv.Tx) error) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return fn(&Tx{bkt: d.root})
}

func (d *DB) Update(fn func(kv.Tx) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return fn(&Tx{bkt: d.root})
}

type Tx struct {
	bkt *Bucket
}

func (t *Tx) Bucket(name []byte) (kv.Bucket, error) {
	return t.bkt.Bucket(name)
}

func (t *Tx) CreateBucketIfNotExists(name []byte) (kv.Bucket, error) {
	return t.bkt.CreateBucketIfNotExists(name)
}

type node struct {
	k, v []byte
}

type Bucket struct {
	name     string
	buckets  map[string]*Bucket
	contents *btree.BTreeG[node]
}

func newBucket(name string) *Bucket {
	return &Bucket{
		name:    name,
		buckets: map[string]*Bucket{},
		contents: btree.NewG(2, func(a, b node) bool {
			// order bytes lexicographically
			return bytes.Compare(a.k, b.k) < 0
		}),
	}
}

func (b *Bucket) Bucket(name []byte) (kv.Bucket, error) {
	if bkt, ok := b.buckets[string(name)]; ok {
		return bkt, nil
	}

	return nil, fmt.Errorf("bucket %q: %w", string(name), kv.ErrNotFound)
}

func (b *Bucket) CreateBucketIfNotExists(name []byte) (kv.Bucket, error) {
	n := string(name)
	if n == "" {
		return nil, errors.New("bucket name cannot be empty")
	}

	bkt, ok := b.buckets[n]
	if !ok {
		bkt = newBucket(n)
		b.buckets[n] = bkt
	}

	return bkt, nil
}

func (b *Bucket) Get(key []byte) ([]byte, error) {
	if node, ok := b.contents.Get(node{k: key}); ok {
		return node.v, nil
	}

	return nil, fmt.Errorf("bucket %q key %q: %w", b.name, string(key), kv.ErrNotFound)
}

func (b *Bucket) Put(k []byte, v []byte) error {
	_, _ = b.contents.ReplaceOrInsert(node{k, v})
	return nil
}

func (b *Bucket) First() (k []byte, v []byte, _ error) {
	if node, ok := b.contents.Min(); ok {
		return node.k, node.v, nil
	}

	return nil, nil, fmt.Errorf("bucket %q first key: %w", b.name, kv.ErrNotFound)
}

func (b *Bucket) Last() (k []byte, v []byte, _ error) {
	if node, ok := b.contents.Max(); ok {
		return node.k, node.v, nil
	}

	return nil, nil, fmt.Errorf("bucket %q last key: %w", b.name, kv.ErrNotFound)
}

func (b *Bucket) Range(opts ...containers.Option[kv.RangeOptions]) iter.Seq2[[]byte, []byte] {
	var options kv.RangeOptions
	containers.ApplyAll(&options, opts...)

	// sorry but here be dragons
	//
	// btree exposes its own iterator interface which is a pain
	// to adapt into the new iter package approach
	// btree forcibly drives its own iteration which means
	// we can't yield when we want in our coro so we use
	// a channel here to push back on the iterator and put our
	// iterator back into the driving seat.

	var (
		ch = make(chan node, 0)
		// stop signals for the btree iterator to return as we no longer
		// want anymore nodes
		stop = make(chan struct{}, 0)
	)

	fn := btree.ItemIteratorG[node](func(n node) bool {
		select {
		case ch <- n:
			return true
		case <-stop:
			return false
		}
	})

	go func() {
		defer close(ch)

		switch options.Order {
		case kv.Descending:
			b.contents.Descend(fn)
		default:
			b.contents.Ascend(fn)
		}
	}()

	return iter.Seq2[[]byte, []byte](func(yield func(k, v []byte) bool) {
		defer close(stop)

		for node := range ch {
			if !yield(node.k, node.v) {
				return
			}
		}
	})
}
