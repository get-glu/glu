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
	"github.com/get-glu/glu/pkg/fs"
	"github.com/get-glu/glu/pkg/phases"
	"github.com/go-git/go-git/v5/plumbing"
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

	_ phases.Source[Resource] = (*Source[Resource])(nil)
)

type Resource interface {
	core.Resource
	ReadFrom(context.Context, core.Metadata, fs.Filesystem) error
	WriteTo(context.Context, core.Metadata, fs.Filesystem) error
}

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

type Source[A Resource] struct {
	mu              sync.RWMutex
	repo            *git.Repository
	proposer        Proposer
	proposeChange   bool
	proposalOptions ProposalOption
}

func (*Source[A]) Metadata() core.Metadata {
	return core.Metadata{
		Name: "git",
	}
}

type ProposalOption struct {
	Labels []string
}

// ProposeChanges configures the phase to propose the change (via PR or MR)
// as opposed to directly integrating it into the target trunk branch.
func ProposeChanges[A Resource](opts ProposalOption) containers.Option[Source[A]] {
	return func(i *Source[A]) {
		i.proposeChange = true
		i.proposalOptions = opts
	}
}

func NewSource[A Resource](repo *git.Repository, proposer Proposer, opts ...containers.Option[Source[A]]) (_ *Source[A]) {
	source := &Source[A]{
		repo:     repo,
		proposer: proposer,
	}

	containers.ApplyAll(source, opts...)

	return source
}

type Branched interface {
	Branch() string
}

func (g *Source[A]) View(ctx context.Context, _, phase core.Metadata, r A) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	opts := []containers.Option[git.ViewUpdateOptions]{}
	if branched, ok := core.Resource(r).(Branched); ok {
		opts = append(opts, git.WithBranch(branched.Branch()))
	}

	return g.repo.View(ctx, func(hash plumbing.Hash, fs fs.Filesystem) error {
		return r.ReadFrom(ctx, phase, fs)
	}, opts...)
}

type commitMessage[A Resource] interface {
	// CommitMessage is an optional git specific method for overriding generated commit messages.
	// The function is provided with the source phases metadata and the previous value of resource.
	CommitMessage(meta core.Metadata, from A) (string, error)
}

type proposalTitle[A Resource] interface {
	// ProposalTitle is an optional git specific method for overriding generated proposal message (PR/MR) title message.
	// The function is provided with the source phases metadata and the previous value of resource.
	ProposalTitle(meta core.Metadata, from A) (string, error)
}

type proposalBody[A Resource] interface {
	// ProposalBody is an optional git specific method for overriding generated proposal body (PR/MR) body message.
	// The function is provided with the source phases metadata and the previous value of resource.
	ProposalBody(meta core.Metadata, from A) (string, error)
}

func (g *Source[A]) Update(ctx context.Context, pipeline, phase core.Metadata, from, to A) (map[string]string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	err := g.repo.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching upstream during update: %w", err)
	}

	// use the target resources branch if it implementes an override
	baseBranch := g.repo.DefaultBranch()
	if branched, ok := core.Resource(to).(Branched); ok {
		baseBranch = branched.Branch()
	}

	annotations := map[string]string{
		AnnotationGitBaseRefKey: baseBranch,
	}

	if !g.proposeChange {
		annotations[AnnotationGitHeadSHAKey], err = g.updateAndPush(ctx, phase, from, to, git.WithBranch(baseBranch))
		if err != nil {
			return nil, err
		}

		return annotations, nil
	}

	if g.proposer == nil {
		return nil, errors.New("proposal requested but not configured")
	}

	return g.propose(ctx, pipeline, phase, from, to, baseBranch)
}

func (g *Source[A]) updateAndPush(ctx context.Context, phase core.Metadata, from, to A, opts ...containers.Option[git.ViewUpdateOptions]) (string, error) {
	slog := slog.With("name", phase.Name)

	update := func(fs fs.Filesystem) (message string, err error) {
		message = fmt.Sprintf("Update %s", phase.Name)
		if m, ok := core.Resource(to).(commitMessage[A]); ok {
			message, err = m.CommitMessage(phase, from)
			if err != nil {
				return "", fmt.Errorf("overriding commit message during update: %w", err)
			}
		}

		if err := to.WriteTo(ctx, phase, fs); err != nil {
			return "", err
		}

		return message, nil
	}

	head, err := g.repo.UpdateAndPush(ctx, update, opts...)
	if err != nil {
		if errors.Is(err, git.ErrEmptyCommit) {
			slog.Debug("promotion produced no changes")

			return "", core.ErrNoChange
		}

		return "", err
	}

	return head.String(), nil
}

func (g *Source[A]) propose(ctx context.Context, pipeline, phase core.Metadata, from, to A, baseBranch string) (map[string]string, error) {
	slog := slog.With("name", phase.Name)

	baseRev, err := g.repo.Resolve(baseBranch)
	if err != nil {
		return nil, fmt.Errorf("resolving base branch %q: %w", baseBranch, err)
	}

	digest, err := to.Digest()
	if err != nil {
		return nil, err
	}

	// create branch name and check if this phase, resource and state has previously been observed
	var (
		branchPrefix = fmt.Sprintf("glu/%s/%s", pipeline.Name, phase.Name)
		branch       = path.Join(branchPrefix, digest)
	)

	// ensure branch exists locally either way
	if err := g.repo.CreateBranchIfNotExists(branch, git.WithBase(baseBranch)); err != nil {
		return nil, err
	}

	proposal, err := g.proposer.GetCurrentProposal(ctx, baseBranch, branchPrefix)
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

		head, err := g.updateAndPush(ctx, phase, from, to, options...)
		if err != nil {
			return nil, fmt.Errorf("updating existing proposal: %w", err)
		}

		// we're updating the head position of an existing proposal
		// so we need to update the value of head in the returned annotations
		annotations := annotations(proposal)
		annotations[AnnotationGitHeadSHAKey] = head

		return annotations, nil
	}

	if head, err := g.updateAndPush(ctx, phase, from, to, options...); err != nil {
		if !errors.Is(err, core.ErrNoChange) {
			return nil, err
		}

		// we check here to see if the update produced no changes due to either
		// the branch previously existing and had been advanced already or
		// the branch was just created and matches the base after update
		// if it is the latter then our update produces no change
		// otherwise, we just need to open the pull request again and continue
		if baseRev.String() == head {
			return nil, core.ErrNoChange
		}
	}

	fromDigest, err := from.Digest()
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf("Update %s", phase.Name)
	if p, ok := core.Resource(to).(proposalTitle[A]); ok {
		title, err = p.ProposalTitle(phase, from)
		if err != nil {
			return nil, err
		}
	}

	body := fmt.Sprintf(`| from | to |
| -- | -- |
| %s | %s |
`, fromDigest, digest)
	if b, ok := core.Resource(to).(proposalBody[A]); ok {
		body, err = b.ProposalBody(phase, from)
		if err != nil {
			return nil, err
		}
	}

	proposal = &Proposal{
		BaseRevision: baseRev.String(),
		BaseBranch:   baseBranch,
		Branch:       branch,
		Title:        title,
		Body:         body,
	}

	if err := g.proposer.CreateProposal(ctx, proposal, g.proposalOptions); err != nil {
		return nil, err
	}

	return annotations(proposal), nil
}

func annotations(proposal *Proposal) map[string]string {
	a := map[string]string{
		AnnotationGitBaseRefKey: proposal.BaseBranch,
		AnnotationGitHeadSHAKey: proposal.HeadRevision,
	}

	maps.Insert(a, maps.All(proposal.Annotations))
	return a
}
