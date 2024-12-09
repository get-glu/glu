package kv

import (
	"errors"
	"iter"

	"github.com/get-glu/glu/pkg/containers"
)

var (
	// ErrNotFound is returned when a read request (get key or bucket) is made
	// but the associated value could not be found
	ErrNotFound = errors.New("not found")
)

// DB is an abstraction around KV database inspired by Bolt
type DB interface {
	View(func(Tx) error) error
	Update(func(Tx) error) error
}

// Tx is an abstraction around KV database transactions
type Tx interface {
	Bucket([]byte) (Bucket, error)
	CreateBucketIfNotExists([]byte) (Bucket, error)
}

// Bucket is an abstraction around KV database buckets
type Bucket interface {
	Bucket([]byte) (Bucket, error)
	CreateBucketIfNotExists([]byte) (Bucket, error)
	Get([]byte) ([]byte, error)
	Put(k, v []byte) error
	First() (k, v []byte, _ error)
	Last() (k, v []byte, _ error)
	Range(opts ...containers.Option[RangeOptions]) iter.Seq2[[]byte, []byte]
}

// Order is a type which identifies a range order
type Order int

const (
	// Ascending represents ascending order
	Ascending Order = iota
	// OrderAsc represents descending order
	Descending
)

// RangeOptions configures a call to Bucket.Range
type RangeOptions struct {
	Order Order
	Start []byte
}

// WithOrder configures a call to Range with the provided order
func WithOrder(o Order) containers.Option[RangeOptions] {
	return func(ro *RangeOptions) {
		ro.Order = o
	}
}

// WithStart configures a call to Range with the provided start key
func WithStart(k []byte) containers.Option[RangeOptions] {
	return func(ro *RangeOptions) {
		ro.Start = k
	}
}
