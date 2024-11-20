package config

import (
	"log/slog"
	"time"
)

var (
	_ validater = (*GitRepositories)(nil)
	_ defaulter = (*GitRepositories)(nil)
)

type GitRepositories map[string]*GitRepository

func (r GitRepositories) validate() error {
	for _, repo := range r {
		if err := repo.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (r GitRepositories) setDefaults() error {
	for _, repo := range r {
		if err := repo.setDefaults(); err != nil {
			return err
		}
	}

	return nil
}

type GitRepository struct {
	Path          string     `glu:"path"`
	DefaultBranch string     `glu:"default_branch"`
	Remote        *Remote    `glu:"remote"`
	Proposals     *Proposals `glu:"proposals"`
}

func (r *GitRepository) validate() error {
	return nil
}

func (r *GitRepository) setDefaults() error {
	if r.DefaultBranch == "" {
		slog.Debug("setting missing default", "repository.default_branch", "main")

		r.DefaultBranch = "main"
	}

	if remote := r.Remote; remote != nil {
		if remote.Name == "" {
			slog.Debug("setting missing default", "repository.remote.name", "origin")

			remote.Name = "origin"
		}
		if remote.Interval == 0 {
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

type Proposals struct {
	Credential string `glu:"credential"`
}
