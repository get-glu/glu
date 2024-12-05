package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"path"
	"sync"

	"github.com/get-glu/glu/internal/git"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/core/typed"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"
)

const (
	AnnotationGitBaseRefKey     = "dev.getglu.git.base_ref"
	AnnotationGitHeadSHAKey     = "dev.getglu.git.head_sha"
	AnnotationProposalNumberKey = "dev.getglu.git.proposal.number"
	AnnotationProposalURLKey    = "dev.getglu.git.proposal.url"
)

var (
	// ErrProposalNotFound is returned when a proposal cannot be located
	ErrProposalNotFound = errors.New("proposal not found")

	_ typed.UpdatablePhase[Resource] = (*Phase[Resource])(nil)
)

// Resource is a core.Resource with additional constraints which are
// required for reading from and writing to a filesystem.
type Resource interface {
	core.Resource
	ReadFrom(context.Context, core.Descriptor, fs.Filesystem) error
	WriteTo(context.Context, core.Descriptor, fs.Filesystem) error
}

// Proposer is a type which can be used to create and manage proposals.
type Proposer interface {
	GetCurrentProposal(_ context.Context, baseBranch, branchPrefix string) (*Proposal, error)
	CreateProposal(context.Context, *Proposal, ProposalOption) error
	MergeProposal(context.Context, *Proposal) error
	CloseProposal(context.Context, *Proposal) error
}

// Proposal contains the fields necessary to propose a resource update
// to a Repository.
type Proposal struct {
	BaseRevision string
	BaseBranch   string
	Branch       string
	HeadRevision string
	Digest       string
	Title        string
	Body         string

	Annotations map[string]string
}

// Phase is a Git storage backed phase implementation.
// It is used to manage the state of a resource as represented in a target Git repository.
type Phase[R Resource] struct {
	mu              sync.RWMutex
	pipeline        string
	meta            core.Metadata
	newFn           func() R
	repo            *git.Repository
	proposer        Proposer
	proposeChange   bool
	proposalOptions ProposalOption
	log             typed.PhaseLogger[R]
}

// Descriptor returns the phases descriptor.
func (p *Phase[A]) Descriptor() core.Descriptor {
	return core.Descriptor{
		Kind:     "git",
		Pipeline: p.pipeline,
		Metadata: p.meta,
	}
}

// ProposalOption configures calls to create proposals
type ProposalOption struct {
	Labels []string
}

// ProposeChanges configures the phase to propose the change (via PR or MR)
// as opposed to directly integrating it into the target trunk branch.
func ProposeChanges[R Resource](opts ProposalOption) containers.Option[Phase[R]] {
	return func(i *Phase[R]) {
		i.proposeChange = true
		i.proposalOptions = opts
	}
}

// WithLogger sets of the reflog on the phase for tracking history
func WithLogger[R Resource](log typed.PhaseLogger[R]) containers.Option[Phase[R]] {
	return func(p *Phase[R]) {
		p.log = log
	}
}

// New constructs and configures a new phase.
func New[R Resource](
	ctx context.Context,
	pipeline string,
	meta core.Metadata,
	newFn func() R,
	repo *git.Repository,
	proposer Proposer,
	opts ...containers.Option[Phase[R]],
) (*Phase[R], error) {
	phase := &Phase[R]{
		pipeline: pipeline,
		meta:     meta,
		newFn:    newFn,
		repo:     repo,
		proposer: proposer,
	}

	containers.ApplyAll(phase, opts...)

	if phase.log != nil {
		if err := phase.log.CreateLog(ctx, phase.Descriptor()); err != nil {
			return nil, err
		}

		phase.repo.Subscribe(phase)
	}

	return phase, nil
}

// Branched is a resource which exposes an alternative base branch
// on which the resource should be based.
type Branched interface {
	Branch(phase core.Descriptor) string
}

func (p *Phase[R]) Get(ctx context.Context) (core.Resource, error) {
	return p.GetResource(ctx)
}

func (p *Phase[R]) GetResource(ctx context.Context) (R, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.getResource(ctx)
}

func (p *Phase[R]) getResource(ctx context.Context) (R, error) {
	var (
		r    = p.newFn()
		desc = p.Descriptor()
	)

	opts := []containers.Option[git.ViewUpdateOptions]{}
	if branched, ok := core.Resource(r).(Branched); ok {
		opts = append(opts, git.WithBranch(branched.Branch(desc)))
	}

	if err := p.repo.View(ctx, func(hash plumbing.Hash, fs fs.Filesystem) error {
		return r.ReadFrom(ctx, desc, fs)
	}, opts...); err != nil {
		return r, err
	}

	return r, nil
}

// History returns the history of the phase (given a log has been configured).
func (p *Phase[R]) History(ctx context.Context) ([]core.State, error) {
	if p.log == nil {
		return nil, nil
	}

	return p.log.History(ctx, p.Descriptor())
}

// Rollback updates the state of the phase to a previous known version in history.
func (p *Phase[R]) Rollback(ctx context.Context, version uuid.UUID) (*core.Result, error) {
	resource, err := p.log.GetResourceAtVersion(ctx, p.Descriptor(), version)
	if err != nil {
		return nil, err
	}

	return p.Update(ctx, resource, typed.UpdateWithKind(typed.KindRollback))
}

func (p *Phase[R]) branch() string {
	if branched, ok := core.Resource(p.newFn()).(Branched); ok {
		return branched.Branch(p.Descriptor())
	}
	return p.repo.DefaultBranch()
}

func (p *Phase[R]) Branches() []string {
	return []string{p.branch()}
}

func (p *Phase[R]) Notify(ctx context.Context, refs map[string]string) error {
	ref, ok := refs[p.branch()]
	if !ok {
		slog.Debug("reference not found on notify", "branch", p.branch(), "refs", refs)
		return nil
	}

	r := p.newFn()
	if err := p.repo.View(ctx, func(hash plumbing.Hash, fs fs.Filesystem) error {
		return r.ReadFrom(ctx, p.Descriptor(), fs)
	}, git.WithRevision(plumbing.NewHash(ref))); err != nil {
		return err
	}

	// record latest
	p.log.RecordLatest(ctx, p.Descriptor(), r, map[string]string{
		AnnotationGitHeadSHAKey: ref,
	})

	return nil
}

func (p *Phase[R]) Update(ctx context.Context, to R, opts ...containers.Option[typed.UpdateOptions]) (*core.Result, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	err := p.repo.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching upstream during update: %w", err)
	}

	from, err := p.getResource(ctx)
	if err != nil {
		return nil, err
	}

	// use the target resources branch if it implementes an override
	desc := p.Descriptor()
	baseBranch := p.repo.DefaultBranch()
	if branched, ok := core.Resource(to).(Branched); ok {
		baseBranch = branched.Branch(desc)
	}

	annotations := map[string]string{
		AnnotationGitBaseRefKey: baseBranch,
	}

	updateOpts := typed.NewUpdateOptions(opts...)
	if !p.proposeChange {
		annotations[AnnotationGitHeadSHAKey], err = p.updateAndPush(ctx, from, to, updateOpts, git.WithBranch(baseBranch))
		if err != nil {
			return nil, err
		}

		return &core.Result{Annotations: annotations}, nil
	}

	if p.proposer == nil {
		return nil, errors.New("proposal requested but not configured")
	}

	annotations, err = p.propose(ctx, from, to, baseBranch, updateOpts)
	if err != nil {
		return nil, err
	}

	return &core.Result{Annotations: annotations}, nil
}

func (p *Phase[R]) propose(ctx context.Context, from, to R, baseBranch string, updateOpts *typed.UpdateOptions) (map[string]string, error) {
	slog := slog.With("name", p.meta.Name)
	desc := p.Descriptor()

	baseRev, err := p.repo.Resolve(baseBranch)
	if err != nil {
		return nil, fmt.Errorf("resolving base branch %q: %w", baseBranch, err)
	}

	fromDigest, err := from.Digest()
	if err != nil {
		return nil, err
	}

	digest, err := to.Digest()
	if err != nil {
		return nil, err
	}

	slog.Debug("proposing update", "from", fromDigest, "to", digest)

	// create branch name and check if this phase, resource and state has previously been observed
	var (
		branchPrefix = fmt.Sprintf("glu/%s/%s", p.pipeline, p.meta.Name)
		branch       = path.Join(branchPrefix, digest)
	)

	// ensure branch exists locally either way
	if err := p.repo.CreateBranchIfNotExists(branch, git.WithBase(baseBranch)); err != nil {
		return nil, err
	}

	proposal, err := p.proposer.GetCurrentProposal(ctx, baseBranch, branchPrefix)
	if err != nil {
		if !errors.Is(err, ErrProposalNotFound) {
			return nil, err
		}

		slog.Debug("proposal not found")
	}

	options := []containers.Option[git.ViewUpdateOptions]{git.WithBranch(branch)}
	if proposal != nil {
		// there is an existing proposal
		slog.Debug("proposal found", "base", proposal.BaseBranch, "base_revision", proposal.BaseRevision)
		if proposal.BaseRevision != baseRev.String() {
			// we're potentially going to force update the branch to move the base
			options = append(options, git.WithForce)
		} else if proposal.Digest == digest {
			// nothing has changed since the last promotion and proposals
			slog.Debug("skipping proposal", "reason", "AlreadyExistsAndUpToDate")

			return annotations(proposal), nil
		}

		head, err := p.updateAndPush(ctx, from, to, updateOpts, options...)
		if err != nil {
			return nil, fmt.Errorf("updating existing proposal: %w", err)
		}

		// we're updating the head position of an existing proposal
		// so we need to update the value of head in the returned annotations
		annotations := annotations(proposal)
		annotations[AnnotationGitHeadSHAKey] = head

		return annotations, nil
	}

	if head, err := p.updateAndPush(ctx, from, to, updateOpts, options...); err != nil {
		slog.Error("while attempting update", "head", head, "error", err)
		return nil, err
	}

	title, err := p.proposalTitle(to, desc, from, updateOpts)
	if err != nil {
		return nil, err
	}

	body, err := p.proposalBody(to, desc, from, updateOpts)
	if err != nil {
		return nil, err
	}

	proposal = &Proposal{
		BaseRevision: baseRev.String(),
		BaseBranch:   baseBranch,
		Branch:       branch,
		Title:        title,
		Body:         body,
	}

	if err := p.proposer.CreateProposal(ctx, proposal, p.proposalOptions); err != nil {
		return nil, err
	}

	return annotations(proposal), nil
}

func (p *Phase[R]) updateAndPush(ctx context.Context, from, to R, updateOpts *typed.UpdateOptions, opts ...containers.Option[git.ViewUpdateOptions]) (string, error) {
	desc := p.Descriptor()
	update := func(fs fs.Filesystem) (message string, err error) {
		if err := to.WriteTo(ctx, desc, fs); err != nil {
			return "", err
		}

		return p.commitMessage(to, desc, from, updateOpts)
	}

	head, err := p.repo.UpdateAndPush(ctx, update, opts...)
	if err != nil {
		if errors.Is(err, git.ErrEmptyCommit) {
			return "", core.ErrNoChange
		}

		return "", err
	}

	return head.String(), nil
}

func annotations(proposal *Proposal) map[string]string {
	a := map[string]string{
		AnnotationGitBaseRefKey: proposal.BaseBranch,
		AnnotationGitHeadSHAKey: proposal.HeadRevision,
	}

	maps.Insert(a, maps.All(proposal.Annotations))
	return a
}

type commitMessageBuilder[R Resource] interface {
	// CommitMessage is an optional git specific method for overriding generated commit messages.
	// The function is provided with the source phases metadata and the previous value of resource.
	CommitMessage(phase core.Descriptor, from R, opts *typed.UpdateOptions) (string, error)
}

func (p *Phase[R]) commitMessage(to core.Resource, phase core.Descriptor, from R, updateOpts *typed.UpdateOptions) (string, error) {
	if m, ok := to.(commitMessageBuilder[R]); ok {
		return m.CommitMessage(phase, from, updateOpts)
	}

	return updateOpts.DefaultMessage(p.Descriptor()), nil
}

type proposalTitleBuilder[R Resource] interface {
	// ProposalTitle is an optional git specific method for overriding generated proposal message (PR/MR) title message.
	// The function is provided with the source phases metadata and the previous value of resource.
	ProposalTitle(phase core.Descriptor, from R, opts *typed.UpdateOptions) (string, error)
}

func (p *Phase[R]) proposalTitle(to core.Resource, phase core.Descriptor, from R, updateOpts *typed.UpdateOptions) (string, error) {
	if p, ok := to.(proposalTitleBuilder[R]); ok {
		return p.ProposalTitle(phase, from, updateOpts)
	}

	return updateOpts.DefaultMessage(p.Descriptor()), nil
}

type proposalBodyBuilder[R Resource] interface {
	// ProposalBody is an optional git specific method for overriding generated proposal body (PR/MR) body message.
	// The function is provided with the source phases metadata and the previous value of resource.
	ProposalBody(phase core.Descriptor, from R, opts *typed.UpdateOptions) (string, error)
}

func (p *Phase[R]) proposalBody(to core.Resource, phase core.Descriptor, from R, updateOpts *typed.UpdateOptions) (string, error) {
	if b, ok := to.(proposalBodyBuilder[R]); ok {
		return b.ProposalBody(phase, from, updateOpts)
	}

	fromDigest, err := from.Digest()
	if err != nil {
		return "", err
	}

	toDigest, err := to.Digest()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`| from | to |
| -- | -- |
| %s | %s |
`, fromDigest, toDigest), nil
}
