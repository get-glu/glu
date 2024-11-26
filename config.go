package glu

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/get-glu/glu/internal/git"
	"github.com/get-glu/glu/internal/oci"
	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/credentials"
	"github.com/get-glu/glu/pkg/scm/github"
	srcgit "github.com/get-glu/glu/pkg/src/git"
	srcoci "github.com/get-glu/glu/pkg/src/oci"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	giturls "github.com/whilp/git-urls"
)

// PipelineBuilder is a utility for extracting configured sources by their name
// derived from glu's conventional configuration format.
type PipelineBuilder[R Resource] struct {
	ctx   context.Context
	conf  *config.Config
	creds *credentials.CredentialSource

	cache struct {
		oci      map[string]*oci.Repository
		repo     map[string]*git.Repository
		proposer map[string]srcgit.Proposer
	}
}

// BuilderFunc returns a pipeline building function suitable for system.AddPipeline.
// It adapts the provided function into the type PipelineBuilder which can be
// passed to both GitSource and OCISource.
// This is a convenience wrapper for typed pipelines.
func BuilderFunc[R Resource](fn func(*PipelineBuilder[R]) (Pipeline, error)) func(ctx context.Context, conf *config.Config) (Pipeline, error) {
	return func(ctx context.Context, conf *config.Config) (Pipeline, error) {
		c := &PipelineBuilder[R]{
			ctx:   ctx,
			conf:  conf,
			creds: credentials.New(conf.Credentials),
		}

		c.cache.oci = map[string]*oci.Repository{}
		c.cache.repo = map[string]*git.Repository{}
		c.cache.proposer = map[string]srcgit.Proposer{}

		return fn(c)
	}
}

// GitSource constructs and configures an instance of a *git.Source
// using the name to lookup the relevant configuration.
// It caches built instances and returns the same instance for subsequent
// calls with the same name.
func GitSource[R srcgit.Resource](b *PipelineBuilder[R], name string, opts ...containers.Option[srcgit.Source[R]]) (_ *srcgit.Source[R], err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("git %q: %w", name, err)
		}
	}()

	// check cache for previously built repository
	if repo, ok := b.cache.repo[name]; ok {
		proposer := b.cache.proposer[name]
		return srcgit.NewSource(repo, proposer, opts...), nil
	}

	conf, ok := b.conf.Sources.Git[name]
	if !ok {
		return nil, errors.New("configuration not found")
	}

	var (
		method  transport.AuthMethod
		srcOpts = []containers.Option[git.Repository]{
			git.WithDefaultBranch(conf.DefaultBranch),
		}
	)

	if conf.Path != "" {
		srcOpts = append(srcOpts, git.WithFilesystemStorage(conf.Path))
	}

	if conf.Remote != nil {
		slog.Debug("configuring remote", "remote", conf.Remote.Name)

		srcOpts = append(srcOpts,
			git.WithRemote(conf.Remote.Name, conf.Remote.URL),
			git.WithInterval(conf.Remote.Interval),
		)

		if conf.Remote.Credential != "" {
			creds, err := b.creds.Get(conf.Remote.Credential)
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

	repo, err := git.NewRepository(b.ctx, slog.Default(), append(srcOpts, git.WithAuth(method))...)
	if err != nil {
		return nil, err
	}

	var proposer srcgit.Proposer
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
			creds, err := b.creds.Get(conf.Proposals.Credential)
			if err != nil {
				return nil, err
			}

			client, err := creds.GitHubClient(b.ctx)
			if err != nil {
				return nil, err
			}

			proposer = github.New(
				client,
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

	b.cache.repo[name] = repo
	b.cache.proposer[name] = proposer

	return srcgit.NewSource(repo, proposer, opts...), nil
}

// OCISource constructs and configures an instance of a *oci.Source
// using the name to lookup the relevant configuration.
// It caches built instances and returns the same instance for subsequent
// calls with the same name.
func OCISource[R srcoci.Resource](b *PipelineBuilder[R], name string) (_ *srcoci.Source[R], err error) {
	// check cache for previously built repository
	if repo, ok := b.cache.oci[name]; ok {
		return srcoci.New[R](repo), nil
	}

	conf, ok := b.conf.Sources.OCI[name]
	if !ok {
		return nil, fmt.Errorf("oci %q: configuration not found", name)
	}

	var cred *credentials.Credential
	if conf.Credential != "" {
		cred, err = b.creds.Get(conf.Credential)
		if err != nil {
			return nil, err
		}
	}

	repo, err := oci.New(conf.Reference, cred)
	if err != nil {
		return nil, err
	}

	b.cache.oci[name] = repo

	return srcoci.New[R](repo), nil
}
