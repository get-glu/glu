package config

import (
	"errors"
	"fmt"
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
	Reference  string
	Credential string
}

func (o *OCIRepository) setDefaults() error {
	return nil
}

func (o *OCIRepository) validate() error {
	if o.Reference == "" {
		return errors.New("field reference is required")
	}

	return nil
}
