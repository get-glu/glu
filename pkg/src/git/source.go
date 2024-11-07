package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/get-glu/glu/internal/git"
	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/controllers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/credentials"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/get-glu/glu/pkg/scm/github"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	giturls "github.com/whilp/git-urls"
)

var _ controllers.Source[Resource] = (*Source[Resource])(nil)

type Resource interface {
	core.Resource
	ReadFrom(context.Context, core.Metadata, fs.Filesystem) error
	WriteTo(context.Context, core.Metadata, fs.Filesystem) error
}

type Source[A Resource] struct {
	name string
	conf *config.Repository

	mu            sync.RWMutex
	source        *git.Repository
	proposer      core.Proposer
	proposeChange bool
	autoMerge     bool
}

// ProposeChanges configures the controller to propose the change (via PR or MR)
// as opposed to directly integrating it into the target trunk branch.
func ProposeChanges[A Resource](i *Source[A]) {
	i.proposeChange = true
}

// AutoMerge configures the proposal to be marked to merge once any conditions are met.
func AutoMerge[A Resource](i *Source[A]) {
	i.autoMerge = true
}

type ConfigSource interface {
	GitRepositoryConfig(name string) (*config.Repository, error)
	GetCredential(name string) (*credentials.Credential, error)
}

func NewSource[A Resource](
	ctx context.Context,
	name string,
	cconf ConfigSource,
	opts ...containers.Option[Source[A]],
) (_ *Source[A], err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("git %q: %w", name, err)
		}
	}()

	conf, err := cconf.GitRepositoryConfig(name)
	if err != nil {
		return nil, err
	}

	var (
		method transport.AuthMethod
		repo   = &Source[A]{
			conf: conf,
			name: name,
		}
		srcOpts = []containers.Option[git.Repository]{}
	)

	containers.ApplyAll(repo, opts...)

	if conf.Path != "" {
		srcOpts = append(srcOpts, git.WithFilesystemStorage(conf.Path))
	}

	if conf.Remote != nil {
		slog.Debug("configuring remote", "remote", conf.Remote.Name)

		srcOpts = append(srcOpts, git.WithRemote(conf.Remote.Name, conf.Remote.URL))

		if conf.Remote.Credential != "" {
			creds, err := cconf.GetCredential(conf.Remote.Credential)
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

	repo.source, err = git.NewRepository(context.Background(), slog.Default(), append(srcOpts, git.WithAuth(method))...)
	if err != nil {
		return nil, err
	}

	if conf.Proposals != nil {
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
		if proposalsEnabled = conf.Proposals.Credential != ""; proposalsEnabled {
			creds, err := cconf.GetCredential(conf.Proposals.Credential)
			if err != nil {
				return nil, err
			}

			client, err := creds.GitHubClient(ctx)
			if err != nil {
				return nil, err
			}

			repo.proposer = github.New(
				client.PullRequests,
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

	return repo, nil
}

type Branched interface {
	Branch() string
}

func (g *Source[A]) getBranch(r core.Resource) string {
	branch := g.conf.DefaultBranch
	if branched, ok := r.(Branched); ok {
		branch = branched.Branch()
	}

	return branch
}

func (g *Source[A]) View(ctx context.Context, meta core.Metadata, r A) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	if err := g.source.Fetch(ctx); err != nil {
		return err
	}

	return g.source.View(ctx, g.getBranch(r), func(hash plumbing.Hash, fs fs.Filesystem) error {
		return r.ReadFrom(ctx, meta, fs)
	})
}

func (g *Source[A]) Update(ctx context.Context, meta core.Metadata, from, to A) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	slog := slog.With("name", meta.Name)

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	if err := g.source.Fetch(ctx); err != nil {
		return err
	}

	message := fmt.Sprintf("Update %s", meta.Name)
	update := func(fs fs.Filesystem) (string, error) {
		if err := to.WriteTo(ctx, meta, fs); err != nil {
			return "", err
		}

		return message, nil
	}

	baseBranch := g.getBranch(to)
	if !g.proposeChange {
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

	if g.proposer == nil {
		return errors.New("proposal requested but not configured")
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
	branch := fmt.Sprintf("glu/%s/%s", meta.Name, digest)
	if _, err := g.source.Resolve(branch); err != nil {
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

	proposal = &core.Proposal{
		BaseRevision: baseRev.String(),
		BaseBranch:   baseBranch,
		Branch:       branch,
		Title:        message,
		Body:         body,
	}

	if err := g.proposer.CreateProposal(ctx, proposal); err != nil {
		return err
	}

	if g.autoMerge {
		return g.proposer.MergeProposal(ctx, proposal)
	}

	return nil
}
