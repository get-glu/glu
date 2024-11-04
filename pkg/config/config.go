package config

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path"

	"github.com/get-glu/glu/internal/config"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Repositories Repositories `glu:"repositories"`
	Credentials  Credentials  `glu:"credentials"`
}

func (c *Config) setDefaults() error {
	if err := c.Repositories.setDefaults(); err != nil {
		return err
	}

	return nil
}

func (c *Config) Validate() error {
	return c.Credentials.validate()
}

func ReadFromPath(configPath string) (_ *Config, err error) {
	encoding := json.Unmarshal
	switch path.Ext(configPath) {
	case ".yaml", ".yml":
		encoding = yaml.Unmarshal
	}

	var fi io.ReadCloser
	fi, err = os.Open(configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		slog.Warn("could not locate glu.yaml")

		fi = io.NopCloser(&nopReadCloser{})
	}

	defer fi.Close()

	decoder := config.NewDecoder[Config](fi, encoding)

	var conf Config
	if err := decoder.Decode(&conf); err != nil {
		return nil, err
	}

	if err := conf.setDefaults(); err != nil {
		return nil, err
	}

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	return &conf, nil
}

type nopReadCloser struct {
}

func (n nopReadCloser) Read(p []byte) (_ int, err error) {
	return 0, io.EOF
}
