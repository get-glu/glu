package config

import (
	"errors"
	"fmt"
)

type Credentials map[string]Credential

func (c Credentials) validate() error {
	for _, cred := range c {
		if err := cred.validate(); err != nil {
			return err
		}
	}

	return nil
}

type CredentialType string

const (
	CredentialTypeBasic       = CredentialType("basic")
	CredentialTypeSSH         = CredentialType("ssh")
	CredentialTypeAccessToken = CredentialType("access_token")
)

type Credential struct {
	Type        CredentialType   `glu:"type"`
	Basic       *BasicAuthConfig `glu:"basic"`
	SSH         *SSHAuthConfig   `glu:"ssh"`
	AccessToken *string          `glu:"access_token"`
}

func (c *Credential) validate() error {
	switch c.Type {
	case CredentialTypeBasic:
		if err := c.Basic.validate(); err != nil {
			return err
		}
	case CredentialTypeSSH:
		if err := c.SSH.validate(); err != nil {
			return err
		}
	case CredentialTypeAccessToken:
		if c.AccessToken == nil || *c.AccessToken == "" {
			return errors.New("field required: access_token")
		}
	default:
		return fmt.Errorf("unexpected credential type %q", c.Type)
	}

	return nil
}

// BasicAuthConfig has configuration for authenticating with private git repositories
// with basic auth.
type BasicAuthConfig struct {
	Username string `glu:"username"`
	Password string `glu:"password"`
}

func (b BasicAuthConfig) validate() error {
	if (b.Username != "" && b.Password == "") || (b.Username == "" && b.Password != "") {
		return errors.New("both username and password need to be provided for basic auth")
	}

	return nil
}

// SSHAuthConfig provides configuration support for SSH private key credentials when
// authenticating with private git repositories
type SSHAuthConfig struct {
	User                  string `glu:"user"`
	Password              string `glu:"password"`
	PrivateKeyBytes       string `glu:"private_key_bytes"`
	PrivateKeyPath        string `glu:"private_key_path"`
	InsecureIgnoreHostKey bool   `glu:"insecure_ignore_host_key"`
}

func (a *SSHAuthConfig) validate() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("ssh authentication: %w", err)
		}
	}()

	if a == nil {
		return errors.New("configuration is mising")
	}

	if a.Password == "" {
		return errors.New("password required")
	}

	if (a.PrivateKeyBytes == "" && a.PrivateKeyPath == "") || (a.PrivateKeyBytes != "" && a.PrivateKeyPath != "") {
		return errors.New("please provide exclusively one of private_key_bytes or private_key_path")
	}

	return nil
}
