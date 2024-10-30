package config

type Repositories map[string]Repository

func (r Repositories) validate() error {
	for _, repo := range r {
		if err := repo.validate(); err != nil {
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

type Remote struct {
	Name       string `glu:"name"`
	URL        string `glu:"url"`
	Credential string `glu:"credential"`
}

type SCM struct {
	Credential string `glu:"credential"`
}
