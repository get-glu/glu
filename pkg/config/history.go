package config

import (
	"fmt"
	"log/slog"
	"os"
)

type BoltDBs map[string]*BoltDB

func (b BoltDBs) validate() error {
	for name, source := range b {
		if err := source.validate(); err != nil {
			return fmt.Errorf("bolt %q: %w", name, err)
		}
	}

	return nil
}

func (b BoltDBs) setDefaults() error {
	for name, source := range b {
		if err := source.setDefaults(name); err != nil {
			return fmt.Errorf("bolt %q: %w", name, err)
		}
	}

	return nil
}

type BoltDB struct {
	Name string `glu:"name"`
	Path string `glu:"path"`
}

func (s *BoltDB) validate() error {
	if s == nil {
		return errFieldRequired("bolt")
	}

	if s.Path == "" {
		return errFieldRequired("path")
	}

	return nil
}

func (s *BoltDB) setDefaults(name string) error {
	if s == nil {
		return nil
	}

	if s.Path == "" {
		fi, err := os.CreateTemp("", "bolt-*.db")
		if err != nil {
			return fmt.Errorf("creating temp dir: %w", err)
		}

		if err := fi.Close(); err != nil {
			return err
		}

		s.Path = fi.Name()

		slog.Info("created temporary file for bolt", "source.bolt", name, "path", s.Path)
	}

	return nil
}
