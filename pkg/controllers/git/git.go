package git

import (
	"context"
	"errors"
	"log/slog"

	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/fs"
)

var ErrProposalNotFound = errors.New("proposal not found")

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

// UpdateOptions are used to carry additional optional update parameters
type UpdateOptions struct {
	// ProposeChange requires the update to be proposed as opposed to
	// being directly integrated into the target repository branch.
	ProposeChange bool
	// AutoMerge causes the proposed change to be automatically merged.
	// This can be combined with SCM branch conditions to ensure that statuses
	// shaved pased as a requirement before the merge is performed.
	// Only effective if ProposeChange is also true.
	AutoMerge bool
}

// Repository is an interface around storage for resources.
// Primarily this is a Git repository.
type Repository interface {
	View(context.Context, Resource) error
	Update(_ context.Context, from, to Resource, _ UpdateOptions) error
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

// Controller is a git backed controller for managing resources.
// It supports reading, writing and proposing changes to Git and any supporting SCM.
type Controller[A any, P interface {
	*A
	Resource
}] struct {
	repo Repository
	meta core.Metadata
	fn   func(core.Metadata) P
	src  Reconciler[P]
	opts UpdateOptions
}

// ControllerOptions is a functional option for configuring a git Source.
type ControllerOptions[A any, P interface {
	*A
	Resource
}] func(*Controller[A, P])

// DependsOn provides the Controller with a dependent Controller.
// On Reconcile the controller will first reconcile the dependency.
// If the target dependency has updated, then it attempts to update
// itself to match this target controller.
func DependsOn[A any, P interface {
	*A
	Resource
}](src Reconciler[P]) ControllerOptions[A, P] {
	return func(i *Controller[A, P]) {
		i.src = src
	}
}

// ProposeChanges configures the controller to propose the change (via PR or MR)
// as opposed to directly integrating it into the target trunk branch.
func ProposeChanges[A any, P interface {
	*A
	Resource
}](i *Controller[A, P]) {
	i.opts.ProposeChange = true
}

// AutoMerge configures the proposal to be marked to merge once any conditions are met.
func AutoMerge[A any, P interface {
	*A
	Resource
}](i *Controller[A, P]) {
	i.opts.AutoMerge = true
}

// Registry is a type which supports the registry of reconciler types.
type Registry interface {
	Register(core.Controller)
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
	opts ...ControllerOptions[A, P]) *Controller[A, P] {

	inst := &Controller[A, P]{repo: repo, meta: meta, fn: fn}
	for _, opt := range opts {
		opt(inst)
	}

	pipeline.Register(inst)

	return inst
}

// Metadata returns the underlying metadata for the resource in the current phase.
func (i *Controller[A, P]) Metadata() core.Metadata {
	return i.meta
}

// Get returns the underlying resource without specific type information.
func (i *Controller[A, P]) Get(ctx context.Context) (any, error) {
	return i.GetResource(ctx)
}

// GetResource returns the identified resource as its concrete pointer type.
func (i *Controller[A, P]) GetResource(ctx context.Context) (P, error) {
	p := i.fn(i.meta)
	if err := i.repo.View(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

// Reconcile forces the controller to retrieve the latest version of the resouce from the underlying repository.
// If a dependent controller has been defined, then it is also reconciled.
// If the dependent controller resource differs, then the controller attempts
// to update its underlying repository to match the new desired state.
func (i *Controller[A, P]) Reconcile(ctx context.Context) error {
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

	// move source phase into destination phase
	to.Metadata().Phase = from.Metadata().Phase

	if err := i.repo.Update(ctx, from, to, i.opts); err != nil {
		return err
	}

	return nil
}
