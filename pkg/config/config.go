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
	Log         Log         `glu:"log"`
	Credentials Credentials `glu:"credentials"`
	Sources     struct {
		Git GitRepositories `glu:"git"`
		OCI OCIRepositories `glu:"oci"`
	} `glu:"sources"`
}

func (c *Config) setDefaults() error {
	if err := c.Log.setDefaults(); err != nil {
		return err
	}

	if err := c.Sources.Git.setDefaults(); err != nil {
		return err
	}

	if err := c.Sources.OCI.setDefaults(); err != nil {
		return err
	}

	return nil
}

func (c *Config) Validate() error {
	if err := c.Log.validate(); err != nil {
		return err
	}

	if err := c.Sources.Git.validate(); err != nil {
		return err
	}

	if err := c.Sources.OCI.validate(); err != nil {
		return err
	}

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
