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
	srcgit "github.com/get-glu/glu/pkg/phases/git"
	"github.com/get-glu/glu/pkg/scm/github"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	giturls "github.com/whilp/git-urls"
)

// Config is a utility for extracting configured sources by their name
// derived from glu's conventional configuration format.
type Config struct {
	ctx   context.Context
	conf  *config.Config
	creds *credentials.CredentialSource

	cache struct {
		oci      map[string]*oci.Repository
		repo     map[string]*git.Repository
		proposer map[string]srcgit.Proposer
	}
}

func newConfigSource(ctx context.Context, conf *config.Config) *Config {
	c := &Config{
		ctx:   ctx,
		conf:  conf,
		creds: credentials.New(conf.Credentials),
	}

	c.cache.oci = map[string]*oci.Repository{}
	c.cache.repo = map[string]*git.Repository{}
	c.cache.proposer = map[string]srcgit.Proposer{}

	return c
}

// GitRepository constructs and configures an instance of a *git.Repository
// using the name to lookup the relevant configuration.
// It caches built instances and returns the same instance for subsequent
// calls with the same name.
func (c *Config) GitRepository(name string) (_ *git.Repository, proposer srcgit.Proposer, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("git %q: %w", name, err)
		}
	}()

	// check cache for previously built repository
	if repo, ok := c.cache.repo[name]; ok {
		return repo, c.cache.proposer[name], nil
	}

	conf, ok := c.conf.Sources.Git[name]
	if !ok {
		return nil, nil, errors.New("configuration not found")
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
			creds, err := c.creds.Get(conf.Remote.Credential)
			if err != nil {
				return nil, nil, fmt.Errorf("repository %q: %w", name, err)
			}

			method, err = creds.GitAuthentication()
			if err != nil {
				return nil, nil, fmt.Errorf("repository %q: %w", name, err)
			}
		}
	}

	if method == nil {
		method, err = ssh.DefaultAuthBuilder("git")
		if err != nil {
			return nil, nil, err
		}
	}

	repo, err := git.NewRepository(c.ctx, slog.Default(), append(srcOpts, git.WithAuth(method))...)
	if err != nil {
		return nil, nil, err
	}

	if conf.Proposals != nil {
		repoURL, err := giturls.Parse(conf.Remote.URL)
		if err != nil {
			return nil, nil, err
		}

		parts := strings.SplitN(strings.TrimPrefix(repoURL.Path, "/"), "/", 2)
		if len(parts) < 2 {
			return nil, nil, fmt.Errorf("unexpected repository URL path: %q", repoURL.Path)
		}

		var (
			repoOwner = parts[0]
			repoName  = strings.TrimSuffix(parts[1], ".git")
		)

		var proposalsEnabled bool
		if proposalsEnabled = conf.Proposals.Credential != ""; proposalsEnabled {
			creds, err := c.creds.Get(conf.Proposals.Credential)
			if err != nil {
				return nil, nil, err
			}

			client, err := creds.GitHubClient(c.ctx)
			if err != nil {
				return nil, nil, err
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

	c.cache.repo[name] = repo
	c.cache.proposer[name] = proposer

	return repo, proposer, nil
}

// OCIRepository constructs and configures an instance of a *oci.Repository
// using the name to lookup the relevant configuration.
// It caches built instances and returns the same instance for subsequent
// calls with the same name.
func (c *Config) OCIRepository(name string) (_ *oci.Repository, err error) {
	// check cache for previously built repository
	if repo, ok := c.cache.oci[name]; ok {
		return repo, nil
	}

	conf, ok := c.conf.Sources.OCI[name]
	if !ok {
		return nil, fmt.Errorf("oci %q: configuration not found", name)
	}

	var cred *credentials.Credential
	if conf.Credential != "" {
		cred, err = c.creds.Get(conf.Credential)
		if err != nil {
			return nil, err
		}
	}

	repo, err := oci.New(conf.Reference, cred)
	if err != nil {
		return nil, err
	}

	c.cache.oci[name] = repo

	return repo, nil
}
