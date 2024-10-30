package glu

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/flipt-io/glu/pkg/config"
	"github.com/flipt-io/glu/pkg/containers"
	"github.com/flipt-io/glu/pkg/fs"
	"github.com/flipt-io/glu/pkg/git"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

var (
	ErrProposalNotFound = errors.New("proposal not found")
)

type Repository interface {
	View(context.Context, *Phase, func(fs.Filesystem) error) error
	Update(context.Context, *Phase, *Metadata, func(fs.Filesystem) (string, error)) error
}

type Proposer interface {
	GetCurrentProposal(context.Context, *Phase, *Metadata) (*Proposal, error)
	CreateProposal(context.Context, *Proposal) error
	CloseProposal(context.Context, *Proposal) error
}

type Proposal struct {
	BaseRevision string
	BaseBranch   string
	Branch       string
	Title        string
	Body         string

	ExternalMetadata map[string]any
}

type GitRepository struct {
	name string
	conf config.Repository

	mu       sync.RWMutex
	source   *git.Source
	proposer Proposer
}

func (p *Pipeline) NewGitRepository(name string) (_ *GitRepository, err error) {
	var method transport.AuthMethod
	method, err = ssh.DefaultAuthBuilder("git")
	if err != nil {
		return nil, err
	}

	opts := []containers.Option[git.Source]{}

	conf, ok := p.conf.Repositories[name]
	if ok {
		if conf.Path != "" {
			opts = append(opts, git.WithFilesystemStorage(conf.Path))
		}

		if conf.Remote != nil {
			slog.Debug("configuring remote", "remote", conf.Remote.Name)

			opts = append(opts, git.WithRemote(conf.Remote.Name, conf.Remote.URL))

			if conf.Remote.Credential != "" {
				creds, err := p.creds.Get(conf.Remote.Credential)
				if err != nil {
					return nil, fmt.Errorf("repository %q: %w", name, err)
				}

				method, err = creds.GitAuthentication()
				if err != nil {
					return nil, fmt.Errorf("repository %q: %w", name, err)
				}
			}
		}
	}

	opts = append(opts, git.WithAuth(method))

	source, err := git.NewSource(context.Background(), slog.Default(), opts...)
	if err != nil {
		return nil, err
	}

	return &GitRepository{
		name:   name,
		source: source,
		conf:   conf,
	}, nil
}

func (g *GitRepository) View(ctx context.Context, phase *Phase, fn func(fs.Filesystem) error) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.source.View(ctx, phase.branch, func(hash plumbing.Hash, fs fs.Filesystem) error {
		return fn(fs)
	})
}

func (g *GitRepository) Update(ctx context.Context, phase *Phase, meta *Metadata, fn func(fs.Filesystem) (string, error)) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	if err := g.source.Fetch(ctx); err != nil {
		return err
	}

	if g.proposer == nil {
		// direct to phase branch without attempting proposals
		if err := g.source.CreateBranchIfNotExists(phase.branch); err != nil {
			return err
		}

		if _, err := g.source.UpdateAndPush(ctx, phase.branch, nil, fn); err != nil {
			if errors.Is(err, git.ErrEmptyCommit) {
				slog.Info("reconcile produced no changes", "phase", phase.name, "name", meta.Name)

				return nil
			}

			return err
		}

		return nil
	}

	rev, err := g.source.Resolve(phase.branch)
	if err != nil {
		return err
	}

	branch := fmt.Sprintf("glu/%s/%s/%s", phase.name, meta.Name, rev)

	proposal, err := g.proposer.GetCurrentProposal(ctx, phase, meta)
	if err != nil && !errors.Is(err, ErrProposalNotFound) {
		return err
	}

	if proposal != nil {
		// there is an existing proposal
		if proposal.BaseRevision == rev.String() {
			// the base revision for the proposal has not changed so
			// we attempt to update the existing proposal with any observed
			// differences
			if err := g.source.CreateBranchIfNotExists(branch, git.WithBase(phase.branch)); err != nil {
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

	if err := g.source.CreateBranchIfNotExists(branch, git.WithBase(phase.branch)); err != nil {
		return err
	}

	if _, err := g.source.UpdateAndPush(ctx, branch, nil, fn); err != nil {
		if errors.Is(err, git.ErrEmptyCommit) {
			slog.Info("reconcile produced no changes", "phase", phase.name, "name", meta.Name)

			return nil
		}

		return err
	}

	if err := g.proposer.CreateProposal(ctx, &Proposal{
		BaseRevision: rev.String(),
		BaseBranch:   phase.branch,
		Branch:       branch,
		Title:        "Some experimental title",
		Body:         "Some experimental body",
	}); err != nil {
		return err
	}

	return nil
}
