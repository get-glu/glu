package config

import (
	"fmt"
)

var (
	_ validater = (*OCISources)(nil)
	_ defaulter = (*OCISources)(nil)
)

type OCISources map[string]*OCIRepository

func (o OCISources) setDefaults() error {
	for name, repo := range o {
		if err := repo.setDefaults(name); err != nil {
			return fmt.Errorf("oci %q: %w", name, err)
		}
	}

	return nil
}

func (o OCISources) validate() error {
	for name, repo := range o {
		if err := repo.validate(); err != nil {
			return fmt.Errorf("oci %q: %w", name, err)
		}
	}

	return nil
}

type OCIRepository struct {
	Name       string `glu:"name"`
	Reference  string `glu:"reference"`
	Credential string `glu:"credential"`
}

func (o *OCIRepository) setDefaults(name string) error {
	if o.Name == "" {
		o.Name = name
	}

	return nil
}

func (o *OCIRepository) validate() error {
	if o.Reference == "" {
		return errFieldRequired("reference")
	}

	return nil
}
