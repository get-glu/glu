package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/credentials"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/get-glu/glu/pkg/git"
	gitsource "github.com/get-glu/glu/pkg/sources/git"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

var _ gitsource.Repository = (*GitRepository)(nil)

type Proposer interface {
	GetCurrentProposal(_ context.Context, baseBranch string, _ *core.Metadata) (*gitsource.Proposal, error)
	CreateProposal(context.Context, *gitsource.Proposal) error
	MergeProposal(context.Context, *gitsource.Proposal) error
	CloseProposal(context.Context, *gitsource.Proposal) error
}

type GitRepository struct {
	conf *config.Repository
	name string

	mu       sync.RWMutex
	source   *git.Source
	proposer Proposer
}

func WithProposer(proposer Proposer) containers.Option[GitRepository] {
	return func(gr *GitRepository) {
		gr.proposer = proposer
	}
}

func NewGitRepository(
	ctx context.Context,
	conf *config.Repository,
	creds *credentials.CredentialSource,
	name string,
	opts ...containers.Option[GitRepository],
) (_ *GitRepository, err error) {
	var (
		method transport.AuthMethod
		repo   = &GitRepository{
			conf: conf,
			name: name,
		}
		srcOpts = []containers.Option[git.Source]{}
	)

	if conf.Path != "" {
		srcOpts = append(srcOpts, git.WithFilesystemStorage(conf.Path))
	}

	if conf.Remote != nil {
		slog.Debug("configuring remote", "remote", conf.Remote.Name)

		srcOpts = append(srcOpts, git.WithRemote(conf.Remote.Name, conf.Remote.URL))

		if conf.Remote.Credential != "" {
			creds, err := creds.Get(conf.Remote.Credential)
			if err != nil {
				return nil, fmt.Errorf("repository %q: %w", name, err)
			}

			method, err = creds.GitAuthentication()
			if err != nil {
				return nil, fmt.Errorf("repository %q: %w", name, err)
			}
		}
	}

	if method == nil {
		method, err = ssh.DefaultAuthBuilder("git")
		if err != nil {
			return nil, err
		}

	}

	repo.source, err = git.NewSource(context.Background(), slog.Default(), append(srcOpts, git.WithAuth(method))...)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

type Branched interface {
	Branch() string
}

func (g *GitRepository) getBranch(r core.Resource) string {
	branch := g.conf.DefaultBranch
	if branched, ok := r.(Branched); ok {
		branch = branched.Branch()
	}

	return branch
}

func (g *GitRepository) View(ctx context.Context, r gitsource.Resource) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	if err := g.source.Fetch(ctx); err != nil {
		return err
	}

	return g.source.View(ctx, g.getBranch(r), func(hash plumbing.Hash, fs fs.Filesystem) error {
		return r.ReadFrom(ctx, fs)
	})
}

func (g *GitRepository) Update(ctx context.Context, from, to gitsource.Resource, opts gitsource.UpdateOptions) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	meta := to.Metadata()

	slog := slog.With("phase", meta.Phase, "name", meta.Name)

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	if err := g.source.Fetch(ctx); err != nil {
		return err
	}

	message := fmt.Sprintf("Update %s in %s", meta.Name, meta.Phase)
	update := func(fs fs.Filesystem) (string, error) {
		if err := to.WriteTo(ctx, fs); err != nil {
			return "", err
		}

		return message, nil
	}

	baseBranch := g.getBranch(to)
	if !opts.ProposeChange {
		// direct to phase branch without attempting proposals
		if err := g.source.CreateBranchIfNotExists(baseBranch); err != nil {
			return err
		}

		if _, err := g.source.UpdateAndPush(ctx, baseBranch, nil, update); err != nil {
			if errors.Is(err, git.ErrEmptyCommit) {
				slog.Info("reconcile produced no changes")

				return nil
			}

			return err
		}

		return nil
	}

	baseRev, err := g.source.Resolve(baseBranch)
	if err != nil {
		return err
	}

	digest, err := to.Digest()
	if err != nil {
		return err
	}

	// create branch name and check if this phase, resource and state has previously been observed
	branch := fmt.Sprintf("glu/%s/%s/%s", meta.Phase, meta.Name, digest)
	if _, err := g.source.Resolve(branch); err != nil {
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			return err
		}
	}

	proposal, err := g.proposer.GetCurrentProposal(ctx, baseBranch, meta)
	if err != nil {
		if !errors.Is(err, gitsource.ErrProposalNotFound) {
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

			if _, err := g.source.UpdateAndPush(ctx, branch, nil, update); err != nil {
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

	if err := g.source.CreateBranchIfNotExists(branch, git.WithBase(baseBranch)); err != nil {
		return err
	}

	if _, err := g.source.UpdateAndPush(ctx, branch, nil, update); err != nil {
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

	proposal = &gitsource.Proposal{
		BaseRevision: baseRev.String(),
		BaseBranch:   baseBranch,
		Branch:       branch,
		Title:        message,
		Body:         body,
	}

	if err := g.proposer.CreateProposal(ctx, proposal); err != nil {
		return err
	}

	if opts.AutoMerge {
		return g.proposer.MergeProposal(ctx, proposal)
	}

	return nil
}
