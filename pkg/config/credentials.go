package config

import (
	"errors"
	"fmt"
)

var _ validate = (*Credentials)(nil)

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
	CredentialTypeGitHubApp   = CredentialType("github_app")
	CredentialTypeDockerLocal = CredentialType("docker_local")
)

type Credential struct {
	Type        CredentialType   `glu:"type"`
	Basic       *BasicAuthConfig `glu:"basic"`
	SSH         *SSHAuthConfig   `glu:"ssh"`
	AccessToken *string          `glu:"access_token"`
	GitHubApp   *GitHubAppConfig `glu:"github_app"`
}

func (c *Credential) validate() error {
	switch c.Type {
	case CredentialTypeBasic:
		return c.Basic.validate()
	case CredentialTypeSSH:
		return c.SSH.validate()
	case CredentialTypeAccessToken:
		if c.AccessToken == nil || *c.AccessToken == "" {
			return errFieldRequired("access_token")
		}
	case CredentialTypeGitHubApp:
		return c.GitHubApp.validate()
	case CredentialTypeDockerLocal:
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

func (b *BasicAuthConfig) validate() error {
	if b == nil || ((b.Username != "" && b.Password == "") || (b.Username == "" && b.Password != "")) {
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
		return errFieldRequired("password")
	}

	if (a.PrivateKeyBytes == "" && a.PrivateKeyPath == "") || (a.PrivateKeyBytes != "" && a.PrivateKeyPath != "") {
		return errors.New("please provide exclusively one of private_key_bytes or private_key_path")
	}

	return nil
}

type GitHubAppConfig struct {
	AppID           int64  `glu:"app_id"`
	InstallationID  int64  `glu:"installation_id"`
	PrivateKeyBytes string `glu:"private_key_bytes"`
	PrivateKeyPath  string `glu:"private_key_path"`
}

func (c *GitHubAppConfig) validate() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("github_app: %w", err)
		}
	}()

	if c.AppID <= 0 {
		return errFieldRequired("app_id")
	}

	if c.InstallationID <= 0 {
		return errFieldRequired("installation_id")
	}

	if (c.PrivateKeyBytes == "" && c.PrivateKeyPath == "") || (c.PrivateKeyBytes != "" && c.PrivateKeyPath != "") {
		return errors.New("please provide exclusively one of private_key_bytes or private_key_path")
	}

	return nil
}
