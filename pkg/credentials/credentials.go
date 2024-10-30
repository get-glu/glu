package credentials

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/flipt-io/glu/pkg/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
)

type CredentialSource struct {
	configs config.Credentials
}

func New(configs config.Credentials) *CredentialSource {
	return &CredentialSource{configs}
}

func (s *CredentialSource) Get(name string) (*Credential, error) {
	config, ok := s.configs[name]
	if !ok {
		return nil, fmt.Errorf("credential not found: %q", name)
	}

	return &Credential{config: &config}, nil
}

type Credential struct {
	config *config.Credential
}

func (c *Credential) HTTPClient(ctx context.Context) (*http.Client, error) {
	switch c.config.Type {
	case config.CredentialTypeBasic:
		return oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
			TokenType: "Basic",
			AccessToken: base64.StdEncoding.EncodeToString([]byte(
				c.config.Basic.Username + ":" + c.config.Basic.Password,
			)),
		})), nil
	case config.CredentialTypeAccessToken:
		return oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
			TokenType:   "Bearer",
			AccessToken: *c.config.AccessToken,
		})), nil
	case config.CredentialTypeSSH:
		return nil, fmt.Errorf("credential type %q not supported for HTTP", c.config.Type)
	}

	return nil, fmt.Errorf("unxpected credential type: %q", c.config.Type)
}

func (c *Credential) GitAuthentication() (auth transport.AuthMethod, err error) {
	switch c.config.Type {
	case config.CredentialTypeBasic:
		return &githttp.BasicAuth{
			Username: c.config.Basic.Username,
			Password: c.config.Basic.Password,
		}, nil
	case config.CredentialTypeSSH:
		conf := c.config.SSH

		if conf.PrivateKeyBytes != "" || conf.PrivateKeyPath != "" {
			var method *gitssh.PublicKeys
			if conf.PrivateKeyBytes != "" {
				method, err = gitssh.NewPublicKeys(
					conf.User,
					[]byte(conf.PrivateKeyBytes),
					conf.Password,
				)
			} else {
				method, err = gitssh.NewPublicKeysFromFile(
					conf.User,
					conf.PrivateKeyPath,
					conf.Password,
				)
			}
			if err != nil {
				return nil, err
			}

			// we're protecting against this explicitly so we can disable
			// the gosec linting rule
			if conf.InsecureIgnoreHostKey {
				// nolint:gosec
				method.HostKeyCallback = ssh.InsecureIgnoreHostKey()
			}

			return method, nil
		}

		// fallback to ssh agent auth for configured user
		return gitssh.DefaultAuthBuilder(conf.User)
	case config.CredentialTypeAccessToken:
		return &githttp.TokenAuth{Token: *c.config.AccessToken}, nil
	}

	return nil, fmt.Errorf("unxpected credential type: %q", c.config.Type)
}
