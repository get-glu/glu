package config

import "log/slog"

type Repositories map[string]*Repository

func (r Repositories) validate() error {
	for _, repo := range r {
		if err := repo.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (r Repositories) setDefaults() error {
	for _, repo := range r {
		if err := repo.setDefaults(); err != nil {
			return err
		}
	}

	return nil
}

type Repository struct {
	Path          string  `glu:"path"`
	DefaultBranch string  `glu:"default_branch"`
	Remote        *Remote `glu:"remote"`
	SCM           *SCM    `glu:"scm"`
}

func (r *Repository) validate() error {
	return nil
}

func (r *Repository) setDefaults() error {
	if r.DefaultBranch == "" {
		slog.Debug("setting missing default", "repository.default_branch", "main")

		r.DefaultBranch = "main"
	}

	if remote := r.Remote; remote != nil {
		if remote.Name == "" {
			slog.Debug("setting missing default", "repository.remote.name", "origin")

			remote.Name = "origin"
		}
	}

	return nil
}

type Remote struct {
	Name       string `glu:"name"`
	URL        string `glu:"url"`
	Credential string `glu:"credential"`
}

type SCM struct {
	Credential string `glu:"credential"`
}
