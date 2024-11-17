package glu

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/get-glu/glu/internal/git"
	"github.com/get-glu/glu/internal/oci"
	"github.com/get-glu/glu/pkg/cli"
	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/credentials"
	"github.com/get-glu/glu/pkg/scm/github"
	srcgit "github.com/get-glu/glu/pkg/src/git"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	giturls "github.com/whilp/git-urls"
	"golang.org/x/sync/errgroup"
)

const defaultScheduleInternal = time.Minute

type Metadata = core.Metadata

type Resource = core.Resource

func Name(name string, opts ...containers.Option[Metadata]) Metadata {
	meta := Metadata{Name: name}
	containers.ApplyAll(&meta, opts...)
	return meta
}

func Label(k, v string) containers.Option[Metadata] {
	return func(m *core.Metadata) {
		if m.Labels == nil {
			m.Labels = map[string]string{}
		}

		m.Labels[k] = v
	}
}

type System struct {
	ctx       context.Context
	meta      Metadata
	conf      *Config
	pipelines map[string]core.Pipeline
	schedules []Schedule
	err       error

	server *Server
}

func NewSystem(ctx context.Context, meta Metadata) *System {
	r := &System{
		ctx:       ctx,
		meta:      meta,
		pipelines: map[string]core.Pipeline{},
	}

	r.server = newServer(r)

	return r
}

func (s *System) GetPipeline(name string) (core.Pipeline, error) {
	pipeline, ok := s.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("pipeline %q: %w", name, core.ErrNotFound)
	}

	return pipeline, nil
}

func (s *System) Pipelines() iter.Seq2[string, core.Pipeline] {
	return maps.All(s.pipelines)
}

func (s *System) AddPipeline(fn func(context.Context, *Config) (core.Pipeline, error)) *System {
	// skip next step if error is not nil
	if s.err != nil {
		return s
	}

	config, err := s.configuration()
	if err != nil {
		s.err = err
		return s
	}

	pipe, err := fn(s.ctx, config)
	if err != nil {
		s.err = err
		return s
	}

	s.pipelines[pipe.Metadata().Name] = pipe
	return s
}

func (s *System) configuration() (_ *Config, err error) {
	if s.conf != nil {
		return s.conf, nil
	}

	conf, err := config.ReadFromPath("glu.yaml")
	if err != nil {
		return nil, err
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(conf.Log.Level)); err != nil {
		return nil, err
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))

	s.conf = newConfigSource(conf)

	return s.conf, nil
}

func (s *System) Run() error {
	if s.err != nil {
		return s.err
	}

	ctx, cancel := signal.NotifyContext(s.ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) > 1 {
		return cli.Run(ctx, s, os.Args...)
	}

	var (
		group errgroup.Group
		srv   = http.Server{
			Addr:    ":8080", // TODO: make configurable
			Handler: s.server,
		}
	)

	group.Go(func() error {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})

	group.Go(func() error {
		slog.Info("starting server", "addr", ":8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cancel()
			return err
		}
		return nil
	})

	group.Go(func() error {
		return s.run(ctx)
	})

	return group.Wait()
}

type Schedule struct {
	interval time.Duration
	options  []containers.Option[core.PhaseOptions]
}

func (s *System) SchedulePromotion(opts ...containers.Option[Schedule]) *System {
	sch := Schedule{
		interval: defaultScheduleInternal,
	}

	containers.ApplyAll(&sch, opts...)

	s.schedules = append(s.schedules, sch)

	return s
}

func ScheduleInterval(d time.Duration) containers.Option[Schedule] {
	return func(s *Schedule) {
		s.interval = d
	}
}

func ScheduleMatchesPhase(c core.Phase) containers.Option[Schedule] {
	return func(s *Schedule) {
		s.options = append(s.options, core.IsPhase(c))
	}
}

func ScheduleMatchesLabel(k, v string) containers.Option[Schedule] {
	return func(s *Schedule) {
		s.options = append(s.options, core.HasLabel(k, v))
	}
}

func (s *System) run(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, sch := range s.schedules {
		wg.Add(1)
		go func(sch Schedule) {
			defer wg.Done()

			ticker := time.NewTicker(sch.interval)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					for _, pipeline := range s.pipelines {
						for phase := range pipeline.Phases(sch.options...) {
							if err := phase.Promote(ctx); err != nil {
								slog.Error("reconciling resource", "name", phase.Metadata().Name, "error", err)
							}
						}
					}

				}
			}
		}(sch)
	}

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		wg.Wait()
	}()

	<-ctx.Done()

	select {
	case <-time.After(15 * time.Second):
		return errors.New("timedout waiting on shutdown of schedules")
	case <-finished:
		return ctx.Err()
	}
}

type Config struct {
	conf  *config.Config
	creds *credentials.CredentialSource

	cache struct {
		oci      map[string]*oci.Repository
		repo     map[string]*git.Repository
		proposer map[string]srcgit.Proposer
	}
}

func newConfigSource(conf *config.Config) *Config {
	c := &Config{
		conf:  conf,
		creds: credentials.New(conf.Credentials),
	}

	c.cache.oci = map[string]*oci.Repository{}
	c.cache.repo = map[string]*git.Repository{}
	c.cache.proposer = map[string]srcgit.Proposer{}

	return c
}

func (c *Config) GitRepository(ctx context.Context, name string) (_ *git.Repository, proposer srcgit.Proposer, err error) {
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

		srcOpts = append(srcOpts, git.WithRemote(conf.Remote.Name, conf.Remote.URL))

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

	repo, err := git.NewRepository(context.Background(), slog.Default(), append(srcOpts, git.WithAuth(method))...)
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

			client, err := creds.GitHubClient(ctx)
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

func (c *Config) GetCredential(name string) (*credentials.Credential, error) {
	return c.creds.Get(name)
}
