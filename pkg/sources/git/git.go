package git

import (
	"context"
	"log/slog"

	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/fs"
)

// Reconciler is a type which expose a Get and Reconcile method
// for a particular Resource type.
type Reconciler[A any] interface {
	GetResource(context.Context) (A, error)
	Reconcile(context.Context) error
}

type Resource interface {
	core.Resource
	ReadFrom(context.Context, fs.Filesystem) error
	WriteTo(context.Context, fs.Filesystem) error
}

// Repository is an interface around storage for resources.
// Primarily this is a Git repository.
type Repository interface {
	View(context.Context, Resource) error
	Update(_ context.Context, from, to Resource) error
}

// Proposal contains the fields necessary to propose a resource update
// to a Repository.
type Proposal struct {
	BaseRevision string
	BaseBranch   string
	Branch       string
	Digest       string
	Title        string
	Body         string

	ExternalMetadata map[string]any
}

type Source[A any, P interface {
	*A
	Resource
}] struct {
	repo Repository
	meta core.Metadata
	fn   func(core.Metadata) P
	src  Reconciler[P]
}

type InstanceOption[A any, P interface {
	*A
	Resource
}] func(*Source[A, P])

func DependsOn[A any, P interface {
	*A
	Resource
}](src Reconciler[P]) InstanceOption[A, P] {
	return func(i *Source[A, P]) {
		i.src = src
	}
}

type Registry interface {
	Register(core.Reconciler)
}

// New constructs and configures a new git Source which can be used
// to fetch and reconcile the state of a resource within a git repository
// with other upstream reconcilable dependencies.
func New[A any, P interface {
	*A
	Resource
}](
	pipeline Registry,
	repo Repository,
	meta core.Metadata,
	fn func(core.Metadata) P,
	opts ...InstanceOption[A, P]) *Source[A, P] {

	inst := &Source[A, P]{repo: repo, meta: meta, fn: fn}
	for _, opt := range opts {
		opt(inst)
	}

	pipeline.Register(inst)

	return inst
}

func (i *Source[A, P]) Metadata() core.Metadata {
	return i.meta
}

func (i *Source[A, P]) Get(ctx context.Context) (any, error) {
	return i.GetResource(ctx)
}

func (i *Source[A, P]) GetResource(ctx context.Context) (P, error) {
	p := i.fn(i.meta)
	if err := i.repo.View(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

func (i *Source[A, P]) Reconcile(ctx context.Context) error {
	slog.Debug("reconcile started", "type", "instance", "phase", i.meta.Phase, "name", i.meta.Name)

	from := i.fn(i.meta)
	if err := i.repo.View(ctx, from); err != nil {
		return err
	}

	if i.src == nil {
		// nothing to reconcile from
		return nil
	}

	if err := i.src.Reconcile(ctx); err != nil {
		return err
	}

	to, err := i.src.GetResource(ctx)
	if err != nil {
		return err
	}

	fromDigest, err := from.Digest()
	if err != nil {
		return err
	}

	toDigest, err := to.Digest()
	if err != nil {
		return err
	}

	if fromDigest == toDigest {
		slog.Debug("skipping reconcile", "reason", "UpToDate")

		return nil
	}

	if err := i.repo.Update(ctx, from, to); err != nil {
		return err
	}

	return nil
}
