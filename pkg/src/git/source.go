package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"sync"

	"github.com/get-glu/glu/internal/git"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/get-glu/glu/pkg/phases"
	"github.com/go-git/go-git/v5/plumbing"
)

var ErrProposalNotFound = errors.New("proposal not found")

var _ phases.Source[Resource] = (*Source[Resource])(nil)

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
	Digest       string
	Title        string
	Body         string

	ExternalMetadata map[string]any
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

func (g *Source[A]) Update(ctx context.Context, pipeline, phase core.Metadata, from, to A) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	slog := slog.With("name", phase.Name)

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	err := g.repo.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetching upstream during update: %w", err)
	}

	message := fmt.Sprintf("Update %s", phase.Name)
	if m, ok := core.Resource(to).(commitMessage[A]); ok {
		message, err = m.CommitMessage(phase, from)
		if err != nil {
			return fmt.Errorf("overriding commit message during update: %w", err)
		}
	}

	update := func(fs fs.Filesystem) (string, error) {
		if err := to.WriteTo(ctx, phase, fs); err != nil {
			return "", err
		}

		return message, nil
	}

	// use the target resources branch if it implementes an override
	baseBranch := g.repo.DefaultBranch()
	if branched, ok := core.Resource(to).(Branched); ok {
		baseBranch = branched.Branch()
	}

	if !g.proposeChange {
		if _, err := g.repo.UpdateAndPush(ctx, update, git.WithBranch(baseBranch)); err != nil {
			if errors.Is(err, git.ErrEmptyCommit) {
				slog.Info("promotion produced no changes")

				return nil
			}

			return err
		}

		return nil
	}

	if g.proposer == nil {
		return errors.New("proposal requested but not configured")
	}

	baseRev, err := g.repo.Resolve(baseBranch)
	if err != nil {
		return fmt.Errorf("resolving base branch %q: %w", baseBranch, err)
	}

	digest, err := to.Digest()
	if err != nil {
		return err
	}

	// create branch name and check if this phase, resource and state has previously been observed
	var (
		branchPrefix = fmt.Sprintf("glu/%s/%s", pipeline.Name, phase.Name)
		branch       = path.Join(branchPrefix, digest)
	)

	// ensure branch exists locally either way
	if err := g.repo.CreateBranchIfNotExists(branch, git.WithBase(baseBranch)); err != nil {
		return err
	}

	proposal, err := g.proposer.GetCurrentProposal(ctx, baseBranch, branchPrefix)
	if err != nil {
		if !errors.Is(err, ErrProposalNotFound) {
			return err
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
		} else {
			if proposal.Digest == digest {
				// nothing has changed since the last promotion and proposals
				slog.Debug("skipping proposal", "reason", "AlreadyExistsAndUpToDate")

				return nil
			}
		}

		if _, err := g.repo.UpdateAndPush(ctx, update, options...); err != nil {
			if errors.Is(err, git.ErrEmptyCommit) {
				slog.Debug("skipping proposal", "reason", "UpdateProducedNoChange")

				return nil
			}

			return fmt.Errorf("updating existing proposal: %w", err)
		}

		return nil
	}

	if _, err := g.repo.UpdateAndPush(ctx, update, options...); err != nil {
		if errors.Is(err, git.ErrEmptyCommit) {
			slog.Info("promotion produced no changes")

			return nil
		}

		return err
	}

	fromDigest, err := from.Digest()
	if err != nil {
		return err
	}

	title := message
	if p, ok := core.Resource(to).(proposalTitle[A]); ok {
		title, err = p.ProposalTitle(phase, from)
		if err != nil {
			return err
		}
	}

	body := fmt.Sprintf(`| from | to |
| -- | -- |
| %s | %s |
`, fromDigest, digest)
	if b, ok := core.Resource(to).(proposalBody[A]); ok {
		body, err = b.ProposalBody(phase, from)
		if err != nil {
			return err
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
		return err
	}

	return nil
}
