package credentials

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/get-glu/glu/pkg/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/go-github/v64/github"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
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

	transport *ghinstallation.Transport
}

func (c *Credential) GitHubClient(ctx context.Context) (_ *github.Client, err error) {
	if c.config.Type == config.CredentialTypeGitHubApp {
		transport, err := c.githubInstallationTransport()
		if err != nil {
			return nil, err
		}

		return github.NewClient(&http.Client{Transport: transport}), nil
	}

	client, err := c.HTTPClient(ctx)
	if err != nil {
		return nil, err
	}

	return github.NewClient(client), nil
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
		return nil, fmt.Errorf("http: credential type %q not supported", c.config.Type)
	}

	return nil, fmt.Errorf("http: unexpected credential type: %q", c.config.Type)
}

func (c *Credential) GitAuthentication() (auth transport.AuthMethod, err error) {
	switch c.config.Type {
	case config.CredentialTypeGitHubApp:
		transport, err := c.githubInstallationTransport()
		if err != nil {
			return nil, err
		}

		return &ghAppInstallation{transport}, nil
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

	return nil, fmt.Errorf("git: unxpected credential type: %q", c.config.Type)
}

func (c *Credential) OCIClient(registry string) (_ *auth.Client, err error) {
	if registry == "docker.io" {
		// it is expected that traffic targeting "docker.io" will be redirected
		// to "registry-1.docker.io"
		// reference: https://github.com/moby/moby/blob/v24.0.0-beta.2/registry/config.go#L25-L48
		registry = "registry-1.docker.io"
	}

	var creds auth.CredentialFunc
	switch c.config.Type {
	case config.CredentialTypeBasic:
		creds = auth.StaticCredential(registry, auth.Credential{
			Username: c.config.Basic.Username,
			Password: c.config.Basic.Password,
		})
	case config.CredentialTypeAccessToken:
		creds = auth.StaticCredential(registry, auth.Credential{
			AccessToken: *c.config.AccessToken,
		})
	case config.CredentialTypeDockerLocal:
		store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
		if err != nil {
			return nil, err
		}

		creds = credentials.Credential(store)
	case config.CredentialTypeGitHubApp:
		transport, err := c.githubInstallationTransport()
		if err != nil {
			return nil, err
		}

		creds = auth.CredentialFunc(func(ctx context.Context, hostport string) (auth.Credential, error) {
			token, err := transport.Token(context.Background())
			if err != nil {
				return auth.EmptyCredential, err
			}

			if hostport == registry {
				return auth.Credential{
					Username: "x-access-token",
					Password: token,
				}, nil
			}

			return auth.EmptyCredential, nil
		})
	default:
		return nil, fmt.Errorf("oci: unxpected credential type: %q", c.config.Type)
	}

	return &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: creds,
	}, nil
}

type ghAppInstallation struct {
	transport *ghinstallation.Transport
}

func (i *ghAppInstallation) String() string {
	return "github-app"
}

func (i *ghAppInstallation) Name() string {
	return i.String()
}

func (i *ghAppInstallation) SetAuth(r *http.Request) {
	token, err := i.transport.Token(r.Context())
	if err != nil {
		slog.Error("Attempting to fetch GitHub app installation token", "error", err)
		return
	}

	r.SetBasicAuth("x-access-token", token)
}

func (c *Credential) githubInstallationTransport() (_ *ghinstallation.Transport, err error) {
	if c.transport != nil {
		return c.transport, nil
	}

	conf := c.config.GitHubApp
	if len(conf.PrivateKeyBytes) > 0 {
		c.transport, err = ghinstallation.New(http.DefaultTransport, conf.AppID, conf.InstallationID, []byte(conf.PrivateKeyBytes))
	} else if conf.PrivateKeyPath != "" {
		c.transport, err = ghinstallation.NewKeyFromFile(http.DefaultTransport, conf.AppID, conf.InstallationID, conf.PrivateKeyPath)
	} else {
		return nil, errors.New("github_app auth: neither private key bytes nor path was provided")
	}
	if err != nil {
		return nil, err
	}

	return c.transport, nil
}
