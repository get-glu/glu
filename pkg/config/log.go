package config

import (
	"fmt"
	"strings"
)

var (
	_ validate  = (*Log)(nil)
	_ defaulter = (*Log)(nil)
)

type Log struct {
	Level string `glu:"level"`
}

func (l *Log) setDefaults() error {
	if l.Level == "" {
		l.Level = "info"
	}

	return nil
}

func (l *Log) validate() error {
	switch strings.ToLower(l.Level) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("unexpected log level: %q", l.Level)
	}
}
