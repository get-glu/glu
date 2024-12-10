package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"path"
	"strings"

	"github.com/get-glu/glu/internal/git"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/core/typed"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/get-glu/glu/pkg/kv/memory"
	"github.com/get-glu/glu/pkg/phases/logger"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"
	giturls "github.com/whilp/git-urls"
)

const (
	AnnotationGitBaseRefKey   = "dev.getglu.git.base_ref"
	AnnotationGitHeadSHAKey   = "dev.getglu.git.head_sha"
	AnnotationGitCommitURLKey = "dev.getglu.git.commit.url"
	AnnotationProposalURLKey  = "dev.getglu.git.proposal.url"
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
	IsProposalOpen(context.Context, *Proposal) bool
	CreateProposal(context.Context, *Proposal, ProposalOption) error
	CloseProposal(context.Context, *Proposal) error
	CommentProposal(context.Context, *Proposal, string) error
}

// Proposal contains the fields necessary to propose a resource update
// to a Repository.
type Proposal struct {
	ID  string
	URL string

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
	pipeline string
	meta     core.Metadata
	newFn    func() R
	repo     *git.Repository
	logger   typed.PhaseLogger[R]

	proposer        Proposer
	proposeChange   bool
	proposalOptions ProposalOption
	currentProposal *Proposal
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
		p.logger = log
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
		// logger defaults to in-memory logger
		logger: logger.New[R](memory.New()),
	}

	containers.ApplyAll(phase, opts...)

	if err := phase.logger.CreateLog(ctx, phase.Descriptor()); err != nil {
		return nil, err
	}

	// subscribe to updates whenever the repo observes
	// an update for the phases associated base branch
	phase.repo.Subscribe(phase)

	// record initial phase state for base branch
	if err := phase.recordPhaseState(ctx, git.WithBranch(phase.branch())); err != nil {
		return nil, err
	}

	// attempt to populate cache with current proposal
	if _, err := phase.getCurrentProposal(ctx); err != nil && !errors.Is(err, ErrProposalNotFound) {
		return nil, err
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
	// we fetch directly from the logger since we always log resource
	// state here on update
	return p.logger.GetLatestResource(ctx, p.Descriptor())
}

// History returns the history of the phase (given a log has been configured).
func (p *Phase[R]) History(ctx context.Context) ([]core.State, error) {
	if p.logger == nil {
		return nil, nil
	}

	return p.logger.History(ctx, p.Descriptor())
}

// Rollback updates the state of the phase to a previous known version in history.
func (p *Phase[R]) Rollback(ctx context.Context, version uuid.UUID) (*core.Result, error) {
	resource, err := p.logger.GetResourceAtVersion(ctx, p.Descriptor(), version)
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

// Notify records the phases state based on the revision of the repository
// as identified in the provided map of branches to revisions.
// This is required and called by repo.Subscribe whenever the phases matching branch
// is updated.
func (p *Phase[R]) Notify(ctx context.Context, refs map[string]string) error {
	ref, ok := refs[p.branch()]
	if !ok {
		slog.Debug("reference not found on notify", "branch", p.branch(), "refs", refs)
		return nil
	}

	return p.recordPhaseState(ctx, git.WithRevision(plumbing.NewHash(ref)))
}

func (p *Phase[R]) recordPhaseState(ctx context.Context, opts ...containers.Option[git.BranchOptions]) (err error) {
	var (
		r    = p.newFn()
		hash plumbing.Hash
	)

	if err := p.repo.View(ctx, func(h plumbing.Hash, fs fs.Filesystem) error {
		hash = h
		return r.ReadFrom(ctx, p.Descriptor(), fs)
	}, opts...); err != nil {
		return err
	}

	annotations := map[string]string{
		AnnotationGitHeadSHAKey: hash.String(),
	}

	p.annotateCommitURL(annotations, hash)

	// record latest
	return p.logger.RecordLatest(ctx, p.Descriptor(), r, annotations)
}

func (p *Phase[R]) annotateCommitURL(annotations map[string]string, hash plumbing.Hash) {
	repoURL, err := giturls.Parse(p.repo.Remote().URLs[0])
	if err != nil {
		slog.Warn("while attempting to parse remote URL", "error", err)
		return
	}

	// TODO: abstract this and support other SCM providers and self-hosted solutions
	if !strings.HasPrefix(repoURL.Host, "github.com") {
		slog.Warn("unsupported remote host", "host", repoURL.Host)
		return
	}

	parts := strings.SplitN(strings.TrimPrefix(repoURL.Path, "/"), "/", 2)
	if len(parts) < 2 {
		slog.Warn("unexpected repository URL path", "path", repoURL.Path)
		return
	}

	var (
		repoOwner = parts[0]
		repoName  = strings.TrimSuffix(parts[1], ".git")
	)

	if repoOwner != "" && repoName != "" {
		annotations[AnnotationGitCommitURLKey] = fmt.Sprintf("https://github.com/%s/%s/commit/%s", repoOwner, repoName, hash)
	}

	return
}

// Update sets the phases state to the provided resource using the resource types
// ReadFrom and WriteTo methods to update state accordingly.
// Given a propose is configured, a proposal will be made and the update will be asynchronous.
func (p *Phase[R]) Update(ctx context.Context, to R, opts ...containers.Option[typed.UpdateOptions]) (*core.Result, error) {
	// inital fetch to ensure we're up to date and avoid conflicts
	if err := p.repo.Fetch(ctx, p.branch()); err != nil {
		return nil, fmt.Errorf("fetching upstream during update: %w", err)
	}

	from, err := p.GetResource(ctx)
	if err != nil {
		return nil, err
	}

	annotations := map[string]string{
		AnnotationGitBaseRefKey: p.branch(),
	}

	updateOpts := typed.NewUpdateOptions(opts...)
	if !p.proposeChange {
		annotations[AnnotationGitHeadSHAKey], err = p.updateAndPush(ctx, from, to, updateOpts, git.WithBranch(p.branch()))
		if err != nil {
			return nil, err
		}

		return &core.Result{Annotations: annotations}, nil
	}

	if p.proposer == nil {
		return nil, errors.New("proposal requested but not configured")
	}

	annotations, err = p.propose(ctx, from, to, updateOpts)
	if err != nil {
		return nil, err
	}

	return &core.Result{Annotations: annotations}, nil
}

func (p *Phase[R]) propose(ctx context.Context, from, to R, updateOpts *typed.UpdateOptions) (map[string]string, error) {
	slog := slog.With("name", p.meta.Name)
	desc := p.Descriptor()

	baseBranch := p.branch()
	baseRev, err := p.repo.Resolve(baseBranch)
	if err != nil {
		return nil, fmt.Errorf("resolving base branch %q: %w", baseBranch, err)
	}

	fromDigest, err := from.Digest()
	if err != nil {
		return nil, err
	}

	toDigest, err := to.Digest()
	if err != nil {
		return nil, err
	}

	slog.Debug("proposing update", "from", fromDigest, "to", toDigest)

	// create branch name and check if this phase, resource and state has previously been observed
	branch := path.Join(p.branchPrefix(), toDigest)

	// ensure branch exists locally either way
	if err := p.repo.CreateBranchIfNotExists(branch, git.WithBase(baseBranch)); err != nil {
		return nil, err
	}

	options := []containers.Option[git.BranchOptions]{
		git.WithBranch(branch),
		git.WithBase(baseBranch),
		git.WithPushIfEmpty,
	}

	makeComment := func(*Proposal) error { return nil }

	proposal, err := p.getCurrentProposal(ctx)
	if err == nil {
		// there is an existing proposal
		slog.Debug("proposal found",
			"base", proposal.BaseBranch, "base_revision", proposal.BaseRevision,
			"proposal_digest", proposal.Digest,
			"destination_digest", toDigest,
		)

		// we're potentially going to force update the branch to move the base
		options = append(options, git.WithForce)

		if proposal.BaseRevision != baseRev.String() {
			head, err := p.updateAndPush(ctx, from, to, updateOpts, options...)
			if err != nil {
				return nil, fmt.Errorf("updating existing proposal: %w", err)
			}

			// we're updating the head position of an existing proposal
			// so we need to update the value of head in the returned annotations
			annotations := annotations(proposal)
			annotations[AnnotationGitHeadSHAKey] = head

			return annotations, nil
		} else if proposal.Digest == toDigest {
			// nothing has changed since the last promotion and proposals
			slog.Debug("skipping proposal", "reason", "AlreadyExistsAndUpToDate")

			return annotations(proposal), nil
		}

		// close the proposal as we're creating a new branch for the new proposal
		if err := p.proposer.CloseProposal(ctx, proposal); err != nil {
			return nil, fmt.Errorf("closing existing proposal: %w", err)
		}

		// configure commenting on the old proposal with a link to the new one
		oldProposal := proposal
		makeComment = func(newProposal *Proposal) error {
			return p.proposer.CommentProposal(ctx, oldProposal, fmt.Sprintf("Closed in favour of new proposal #%v", newProposal.ID))
		}
	} else if !errors.Is(err, ErrProposalNotFound) {
		return nil, err
	}

	slog.Debug("proposal not found")

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
		Digest:       toDigest,
		Title:        title,
		Body:         body,
	}

	if err := p.proposer.CreateProposal(ctx, proposal, p.proposalOptions); err != nil {
		return nil, err
	}

	// set current proposal
	p.currentProposal = proposal

	return annotations(proposal), makeComment(proposal)
}

func (p *Phase[R]) getCurrentProposal(ctx context.Context) (*Proposal, error) {
	if p.proposer == nil {
		return nil, ErrProposalNotFound
	}

	if p.currentProposal != nil {
		if !p.proposer.IsProposalOpen(ctx, p.currentProposal) {
			p.currentProposal = nil

			return nil, ErrProposalNotFound
		}

		return p.currentProposal, nil
	}

	proposal, err := p.proposer.GetCurrentProposal(ctx, p.branch(), p.branchPrefix())
	if err != nil {
		return nil, err
	}

	p.currentProposal = proposal

	return proposal, nil
}

func (p *Phase[R]) branchPrefix() string {
	return fmt.Sprintf("glu/%s/%s", p.pipeline, p.meta.Name)
}

func (p *Phase[R]) updateAndPush(ctx context.Context, from, to R, updateOpts *typed.UpdateOptions, opts ...containers.Option[git.BranchOptions]) (string, error) {
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
		AnnotationProposalURLKey: proposal.URL,
		AnnotationGitBaseRefKey:  proposal.BaseBranch,
		AnnotationGitHeadSHAKey:  proposal.HeadRevision,
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
