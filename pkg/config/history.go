package config

import (
	"fmt"
	"log/slog"
	"os"
)

type HistorySources map[string]*HistorySource

func (s HistorySources) validate() error {
	for name, source := range s {
		if err := source.validate(); err != nil {
			return fmt.Errorf("history %q: %w", name, err)
		}
	}

	return nil
}

func (s HistorySources) setDefaults() error {
	for name, source := range s {
		if err := source.setDefaults(name); err != nil {
			return fmt.Errorf("history %q: %w", name, err)
		}
	}

	return nil
}

type HistorySourceType string

const (
	HistorySourceTypeBoltDB = "bolt"
)

type HistorySource struct {
	Name string            `glu:"name"`
	Type HistorySourceType `glu:"type"`
	Path string            `glu:"path"`
}

func (s *HistorySource) validate() error {
	if s == nil {
		return errFieldRequired("history")
	}

	switch s.Type {
	case HistorySourceTypeBoltDB:
		return nil
	}

	return fmt.Errorf("unexpected history source type: %q", s.Type)
}

func (s *HistorySource) setDefaults(name string) error {
	if s == nil {
		return nil
	}

	switch s.Type {
	case "", HistorySourceTypeBoltDB:
		if s.Type == "" {
			slog.Debug("setting missing default", "source.history.type", HistorySourceTypeBoltDB)

			s.Type = HistorySourceTypeBoltDB
		}

		if s.Path == "" {
			fi, err := os.CreateTemp("", "history-*.db")
			if err != nil {
				return fmt.Errorf("creating temp dir: %w", err)
			}

			if err := fi.Close(); err != nil {
				return err
			}

			s.Path = fi.Name()

			slog.Info("created temporary file for history", "source.history", name, "path", s.Path)
		}
	}

	return nil
}
