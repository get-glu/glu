package config

import (
	"fmt"
	"log/slog"
	"os"
)

type HistoryType string

const (
	HistoryTypeFile HistoryType = "file"
)

type History struct {
	Type HistoryType `glu:"type"`
	File FileDBs     `glu:"file"`
}

func (h *History) setDefaults() error {
	if h.Type == "" {
		h.Type = HistoryTypeFile
	}

	return nil
}

type FileDBs map[string]*FileDB

func (b FileDBs) validate() error {
	for name, source := range b {
		if err := source.validate(); err != nil {
			return fmt.Errorf("file db %q: %w", name, err)
		}
	}

	return nil
}

func (b FileDBs) setDefaults() error {
	for name, source := range b {
		if err := source.setDefaults(name); err != nil {
			return fmt.Errorf("file db %q: %w", name, err)
		}
	}

	return nil
}

type FileDB struct {
	Name string `glu:"name"`
	Path string `glu:"path"`
}

func (s *FileDB) validate() error {
	if s == nil {
		return errFieldRequired("file")
	}

	if s.Path == "" {
		return errFieldRequired("path")
	}

	return nil
}

func (s *FileDB) setDefaults(name string) error {
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

		slog.Info("created temporary file for file db", "history.file", name, "path", s.Path)
	}

	return nil
}
