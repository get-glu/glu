package config

import (
	"fmt"
)

var (
	_ validater = (*OCIRepositories)(nil)
	_ defaulter = (*OCIRepositories)(nil)
)

type OCIRepositories map[string]*OCIRepository

func (o OCIRepositories) setDefaults() error {
	for name, repo := range o {
		if err := repo.setDefaults(); err != nil {
			return fmt.Errorf("oci %q: %w", name, err)
		}
	}

	return nil
}

func (o OCIRepositories) validate() error {
	for name, repo := range o {
		if err := repo.validate(); err != nil {
			return fmt.Errorf("oci %q: %w", name, err)
		}
	}

	return nil
}

type OCIRepository struct {
	Reference  string `glu:"reference"`
	Credential string `glu:"credential"`
}

func (o *OCIRepository) setDefaults() error {
	return nil
}

func (o *OCIRepository) validate() error {
	if o.Reference == "" {
		return errFieldRequired("reference")
	}

	return nil
}
