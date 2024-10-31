package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"

	"github.com/flipt-io/glu/pkg/config"
	"github.com/flipt-io/glu/pkg/containers"
	"github.com/flipt-io/glu/pkg/core"
	"github.com/flipt-io/glu/pkg/credentials"
	"github.com/flipt-io/glu/pkg/fs"
	"github.com/flipt-io/glu/pkg/git"
	githubscm "github.com/flipt-io/glu/pkg/scm/github"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/go-github/v64/github"
)

type Proposer interface {
	GetCurrentProposal(context.Context, *core.Phase, *core.Metadata) (*core.Proposal, error)
	CreateProposal(context.Context, *core.Proposal) error
	CloseProposal(context.Context, *core.Proposal) error
}

type GitRepository struct {
	name string
	conf config.Repository

	mu       sync.RWMutex
	source   *git.Source
	proposer Proposer
}

func NewGitRepository(ctx context.Context, conf config.Repository, creds *credentials.CredentialSource, name string) (_ *GitRepository, err error) {
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
		repoURL, err := url.Parse(conf.Remote.URL)
		if err != nil {
			return nil, err
		}

		parts := strings.SplitN(strings.TrimPrefix(repoURL.Path, "/"), "/", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("unexpected repository URL path: %q", repoURL.Path)
		}

		var (
			ghClient  = github.NewClient(nil)
			repoOwner = parts[0]
			repoName  = strings.TrimSuffix(parts[1], ".git")
		)

		if conf.SCM.Credential != "" {
			creds, err := creds.Get(conf.SCM.Credential)
			if err != nil {
				return nil, fmt.Errorf("repository %q: %w", name, err)
			}

			client, err := creds.HTTPClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("repository %q: %w", name, err)
			}

			ghClient = github.NewClient(client)
			proposer = githubscm.New(ghClient.PullRequests, repoOwner, repoName)
		}
	}

	opts = append(opts, git.WithAuth(method))

	source, err := git.NewSource(context.Background(), slog.Default(), opts...)
	if err != nil {
		return nil, err
	}

	return &GitRepository{
		name:     name,
		source:   source,
		conf:     conf,
		proposer: proposer,
	}, nil
}

func (g *GitRepository) View(ctx context.Context, phase *core.Phase, fn func(fs.Filesystem) error) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.source.View(ctx, phase.Branch(), func(hash plumbing.Hash, fs fs.Filesystem) error {
		return fn(fs)
	})
}

func (g *GitRepository) Update(ctx context.Context, phase *core.Phase, meta *core.Metadata, fn func(fs.Filesystem) (string, error)) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	if err := g.source.Fetch(ctx); err != nil {
		return err
	}

	if g.proposer == nil {
		// direct to phase branch without attempting proposals
		if err := g.source.CreateBranchIfNotExists(phase.Branch()); err != nil {
			return err
		}

		if _, err := g.source.UpdateAndPush(ctx, phase.Branch(), nil, fn); err != nil {
			if errors.Is(err, git.ErrEmptyCommit) {
				slog.Info("reconcile produced no changes", "phase", phase.Name(), "name", meta.Name)

				return nil
			}

			return err
		}

		return nil
	}

	rev, err := g.source.Resolve(phase.Branch())
	if err != nil {
		return err
	}

	branch := fmt.Sprintf("glu/%s/%s/%s", phase.Name(), meta.Name, rev)

	proposal, err := g.proposer.GetCurrentProposal(ctx, phase, meta)
	if err != nil && !errors.Is(err, githubscm.ErrProposalNotFound) {
		return err
	}

	if proposal != nil {
		// there is an existing proposal
		if proposal.BaseRevision == rev.String() {
			// the base revision for the proposal has not changed so
			// we attempt to update the existing proposal with any observed
			// differences
			if err := g.source.CreateBranchIfNotExists(branch, git.WithBase(phase.Branch())); err != nil {
				return err
			}

			if _, err := g.source.UpdateAndPush(ctx, branch, nil, fn); err != nil {
				if errors.Is(err, git.ErrEmptyCommit) {
					slog.Debug("skipping proposal", "reason", "AlreadyExistsAndUpToDate")

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

	if err := g.source.CreateBranchIfNotExists(branch, git.WithBase(phase.Branch())); err != nil {
		return err
	}

	if _, err := g.source.UpdateAndPush(ctx, branch, nil, fn); err != nil {
		if errors.Is(err, git.ErrEmptyCommit) {
			slog.Info("reconcile produced no changes", "phase", phase.Name(), "name", meta.Name)

			return nil
		}

		return err
	}

	if err := g.proposer.CreateProposal(ctx, &core.Proposal{
		BaseRevision: rev.String(),
		BaseBranch:   phase.Branch(),
		Branch:       branch,
		Title:        "Some experimental title",
		Body:         "Some experimental body",
	}); err != nil {
		return err
	}

	return nil
}
