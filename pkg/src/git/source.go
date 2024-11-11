package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/get-glu/glu/internal/git"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/controllers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/go-git/go-git/v5/plumbing"
)

var _ controllers.Source[Resource] = (*Source[Resource])(nil)

type Resource interface {
	core.Resource
	ReadFrom(context.Context, core.Metadata, fs.Filesystem) error
	WriteTo(context.Context, core.Metadata, fs.Filesystem) error
}

type Proposer interface {
	GetCurrentProposal(_ context.Context, _ core.Metadata, baseBranch string) (*Proposal, error)
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

type ProposalOption struct {
	Labels []string
}

// ProposeChanges configures the controller to propose the change (via PR or MR)
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

func (g *Source[A]) View(ctx context.Context, meta core.Metadata, r A) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	if err := g.repo.Fetch(ctx); err != nil {
		return err
	}

	opts := []containers.Option[git.ViewUpdateOptions]{}
	if branched, ok := core.Resource(r).(Branched); ok {
		opts = append(opts, git.WithBranch(branched.Branch()))
	}

	return g.repo.View(ctx, func(hash plumbing.Hash, fs fs.Filesystem) error {
		return r.ReadFrom(ctx, meta, fs)
	}, opts...)
}

func (g *Source[A]) Update(ctx context.Context, meta core.Metadata, from, to A) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	slog := slog.With("name", meta.Name)

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	if err := g.repo.Fetch(ctx); err != nil {
		return err
	}

	message := fmt.Sprintf("Update %s", meta.Name)
	update := func(fs fs.Filesystem) (string, error) {
		if err := to.WriteTo(ctx, meta, fs); err != nil {
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
				slog.Info("reconcile produced no changes")

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
		return err
	}

	digest, err := to.Digest()
	if err != nil {
		return err
	}

	// create branch name and check if this phase, resource and state has previously been observed
	branch := fmt.Sprintf("glu/%s/%s", meta.Name, digest)
	if _, err := g.repo.Resolve(branch); err != nil {
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			return err
		}
	}

	proposal, err := g.proposer.GetCurrentProposal(ctx, meta, baseBranch)
	if err != nil {
		if !errors.Is(err, core.ErrProposalNotFound) {
			return err
		}

		slog.Debug("proposal not found")
	}

	if proposal != nil {
		// there is an existing proposal
		if proposal.BaseRevision == baseRev.String() {
			if proposal.Digest == digest {
				// nothing has changed since the last reconciliation and proposals
				slog.Debug("skipping proposal", "reason", "AlreadyExistsAndUpToDate")

				return nil
			}

			if _, err := g.repo.UpdateAndPush(ctx, update, git.WithBranch(branch)); err != nil {
				if errors.Is(err, git.ErrEmptyCommit) {
					slog.Debug("skipping proposal", "reason", "UpdateProducedNoChange")

					return nil
				}

				return err
			}

			// existing proposal has been updated

			return nil
		}

		// current open proposal is based on an outdated revision
		// so we're going to close this PR and create a new one from
		// the new base
		if err := g.proposer.CloseProposal(ctx, proposal); err != nil {
			return err
		}
	}

	if err := g.repo.CreateBranchIfNotExists(branch, git.WithBase(baseBranch)); err != nil {
		return err
	}

	if _, err := g.repo.UpdateAndPush(ctx, update, git.WithBranch(branch)); err != nil {
		if errors.Is(err, git.ErrEmptyCommit) {
			slog.Info("reconcile produced no changes")

			return nil
		}

		return err
	}

	fromDigest, err := from.Digest()
	if err != nil {
		return err
	}

	body := fmt.Sprintf(`%s:
| app | from | to |
| --- | ---- | -- |
| %s | %s | %s |
`, message, meta.Name, fromDigest, digest)

	proposal = &Proposal{
		BaseRevision: baseRev.String(),
		BaseBranch:   baseBranch,
		Branch:       branch,
		Title:        message,
		Body:         body,
	}

	if err := g.proposer.CreateProposal(ctx, proposal, g.proposalOptions); err != nil {
		return err
	}

	return nil
}
