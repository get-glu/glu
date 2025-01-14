package config

import (
	"fmt"
	"log/slog"
	"time"
)

var (
	_ validater = (*GitSources)(nil)
	_ defaulter = (*GitSources)(nil)
)

type GitSources map[string]*GitRepository

func (r GitSources) validate() error {
	for name, repo := range r {
		if err := repo.validate(); err != nil {
			return fmt.Errorf("git %q: %w", name, err)
		}
	}

	return nil
}

func (r GitSources) setDefaults() error {
	for name, repo := range r {
		if err := repo.setDefaults(name); err != nil {
			return fmt.Errorf("git %q: %w", name, err)
		}
	}

	return nil
}

type GitRepository struct {
	Name          string  `glu:"name"`
	Path          string  `glu:"path"`
	DefaultBranch string  `glu:"default_branch"`
	Remote        *Remote `glu:"remote"`
}

func (r *GitRepository) validate() error {
	if r.DefaultBranch == "" {
		return errFieldRequired("default_branch")
	}

	if remote := r.Remote; remote != nil {
		if remote.Name == "" {
			return errFieldRequired("remote.name")
		}

		if remote.URL == "" {
			return errFieldRequired("remote.url")
		}

		if remote.Interval < 1 {
			return errFieldPositiveNonZero("remote.interval")
		}
	}

	return nil
}

func (r *GitRepository) setDefaults(name string) error {
	if r.DefaultBranch == "" {
		slog.Debug("setting missing default", "source.git", name, "default_branch", "main")

		r.DefaultBranch = "main"
	}

	if remote := r.Remote; remote != nil {
		if remote.Name == "" {
			slog.Debug("setting missing default", "source.git", name, "remote.name", "origin")

			remote.Name = "origin"
		}

		if remote.Interval < 1 {
			remote.Interval = 10 * time.Second
		}
	}

	return nil
}

type Remote struct {
	Name       string        `glu:"name"`
	URL        string        `glu:"url"`
	Credential string        `glu:"credential"`
	Interval   time.Duration `glu:"interval"`
}
