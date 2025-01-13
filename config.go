package glu

import (
	"context"

	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/credentials"
)

// Config is a utility for extracting configured sources by their name
// derived from glu's conventional configuration format.
type Config struct {
	ctx   context.Context
	conf  *config.Config
	creds *credentials.CredentialSource
}

func newConfigSource(ctx context.Context, conf *config.Config) *Config {
	return &Config{
		ctx:   ctx,
		conf:  conf,
		creds: credentials.New(conf.Credentials),
	}
}
