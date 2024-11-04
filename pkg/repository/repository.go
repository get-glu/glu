package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/credentials"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/get-glu/glu/pkg/git"
	githubscm "github.com/get-glu/glu/pkg/scm/github"
	gitsource "github.com/get-glu/glu/pkg/sources/git"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/go-github/v64/github"
	giturls "github.com/whilp/git-urls"
)

var _ gitsource.Repository = (*GitRepository)(nil)

type Proposer interface {
	GetCurrentProposal(_ context.Context, baseBranch string, _ *core.Metadata) (*gitsource.Proposal, error)
	CreateProposal(context.Context, *gitsource.Proposal) error
	CloseProposal(context.Context, *gitsource.Proposal) error
}

type GitRepository struct {
	conf *config.Repository

	name            string
	enableProposals bool

	mu       sync.RWMutex
	source   *git.Source
	proposer Proposer
}

func NewGitRepository(ctx context.Context, conf *config.Repository, creds *credentials.CredentialSource, name string, enableProposals bool) (_ *GitRepository, err error) {
	var method transport.AuthMethod
	method, err = ssh.DefaultAuthBuilder("git")
	if err != nil {
		return nil, err
	}

	var (
		opts     = []containers.Option[git.Source]{}
		proposer Proposer
	)

	if conf.Path != "" {
		opts = append(opts, git.WithFilesystemStorage(conf.Path))
	}

	if conf.Remote != nil {
		slog.Debug("configuring remote", "remote", conf.Remote.Name)

		opts = append(opts, git.WithRemote(conf.Remote.Name, conf.Remote.URL))

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

	if conf.SCM != nil {
		repoURL, err := giturls.Parse(conf.Remote.URL)
		if err != nil {
			return nil, err
		}

		parts := strings.SplitN(strings.TrimPrefix(repoURL.Path, "/"), "/", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("unexpected repository URL path: %q", repoURL.Path)
		}

		var (
			repoOwner = parts[0]
			repoName  = strings.TrimSuffix(parts[1], ".git")
		)

		var proposalsEnabled bool
		if proposalsEnabled = conf.SCM.Credential != ""; proposalsEnabled {
			creds, err := creds.Get(conf.SCM.Credential)
			if err != nil {
				return nil, fmt.Errorf("repository %q: %w", name, err)
			}

			client, err := creds.HTTPClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("repository %q: %w", name, err)
			}

			proposer = githubscm.New(
				github.NewClient(client).PullRequests,
				repoOwner,
				repoName,
			)
		}

		slog.Debug("configured scm proposer",
			slog.String("owner", repoOwner),
			slog.String("name", repoName),
			slog.Bool("proposals_enabled", proposalsEnabled),
		)
	}

	opts = append(opts, git.WithAuth(method))

	source, err := git.NewSource(context.Background(), slog.Default(), opts...)
	if err != nil {
		return nil, err
	}

	return &GitRepository{
		conf:            conf,
		name:            name,
		enableProposals: enableProposals,
		source:          source,
		proposer:        proposer,
	}, nil
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

func (g *GitRepository) Update(ctx context.Context, from, to gitsource.Resource) error {
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
	if !g.enableProposals {
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
		if !errors.Is(err, githubscm.ErrProposalNotFound) {
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

	if err := g.proposer.CreateProposal(ctx, &gitsource.Proposal{
		BaseRevision: baseRev.String(),
		BaseBranch:   baseBranch,
		Branch:       branch,
		Title:        message,
		Body:         body,
	}); err != nil {
		return err
	}

	return nil
}
