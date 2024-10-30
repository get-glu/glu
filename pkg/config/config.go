package config

import (
	"encoding/json"
	"os"
	"path"

	"github.com/flipt-io/glu/internal/config"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Repositories Repositories `glu:"repositories"`
	Credentials  Credentials  `glu:"credentials"`
}

func (c *Config) Validate() error {
	return c.Credentials.validate()
}

func ReadFromPath(configPath string) (*Config, error) {
	encoding := json.Unmarshal
	switch path.Ext(configPath) {
	case "yaml", "yml":
		encoding = yaml.Unmarshal
	}

	fi, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}

	defer fi.Close()

	decoder := config.NewDecoder[Config](fi, encoding)

	var conf Config
	if err := decoder.Decode(&conf); err != nil {
		return nil, err
	}

	return &conf, nil
}
