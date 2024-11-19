# Configuration

This section outlines the configuration options available for Glu.

Glu can be configured in a number of ways:

- [Configuration File](#configuration-file)
- [Environment Variables](#environment-variables)

## Configuration File

The configuration file is a YAML file (`glu.yaml` or `glu.yml`) that contains the configuration for Glu.

It's structure looks like this:

```yaml
log:
  level: info

credentials:
  github: # name of the credential to be referenced in the sources
    type: basic
    basic:
      username: "your_username"
      password: "your_personal_access_token"

sources:
  git:
    example: # name of the source to be referenced in glu
    remote:
      name: origin
      url: https://github.com/get-glu/example-app
      credential: github # name of the credential (above) to be used for this source
```

Credentials are used to authenticate with the source. [Sources](/?id=sources) allow for viewing and updating of resources (e.g. git repositories, OCI images).

## Environment Variables

All configuration options can also be set using environment variables using the `GLU_` prefix.

Environment variables **take precedence** over the configuration file.

Keys in the environment variable must be in uppercase and use underscores (`_`) instead of dots (`.`).

For example, `log.level` in the configuration file would be represented as `GLU_LOG_LEVEL`.

```bash
export GLU_LOG_LEVEL=info
```

## Values

The following values can be set in the configuration file or environment variables.

### log

#### `log.level`

The logging level to use.

Valid values are `debug`, `info`, `warn`, and `error`.

### credentials

#### `credentials.<name>`

The configuration for a named credential.

#### `credentials.<name>.type`

The type of credential.

Valid values are:

- `basic`
- `ssh`
- `access_token`
- `github_app`
- `docker_local`

#### credentials.\<name\>.basic

The configuration for basic authentication.

#### `credentials.<name>.basic.username`

The username to use for basic authentication.

#### `credentials.<name>.basic.password`

The password to use for basic authentication'.

#### credentials.\<name\>.ssh

The configuration for SSH authentication.

#### `credentials.<name>.ssh.user`

The user to use for SSH authentication.

#### `credentials.<name>.ssh.password`

The password to use for SSH authentication.

#### `credentials.<name>.ssh.private_key_bytes`

The private key to use for SSH authentication as a PEM encoded string.

#### `credentials.<name>.ssh.private_key_path`

The path to the private key to use for SSH authentication.

#### `credentials.<name>.ssh.insecure_ignore_host_key`

Whether to ignore the host key for SSH authentication.

**Not recommended for production use.**

#### credentials.\<name\>.access_token

The access token to use for authentication.

#### credentials.\<name\>.github_app

The configuration for GitHub App authentication.

#### `credentials.<name>.github_app.app_id`

The ID of the GitHub App.

#### `credentials.<name>.github_app.installation_id`

The installation ID of the GitHub App.

#### `credentials.<name>.github_app.private_key_bytes`

The private key of the GitHub App as a PEM encoded string.

#### `credentials.<name>.github_app.private_key_path`

The path to the private key of the GitHub App.

### sources

#### `sources.<name>`

The configuration for a named source.

#### sources.\<name\>.git

The configuration for a git source.

#### `sources.<name>.git.<repository>`

The configuration for a git repository.

#### `sources.<name>.git.<repository>.path`

The path to the git repository on the local filesystem.

#### `sources.<name>.git.<repository>.default_branch`

The default branch to use for the git repository.

#### `sources.<name>.git.<repository>.remote`

The configuration for the remote of the git repository.

#### `sources.<name>.git.<repository>.remote.name`

The name of the remote.

#### `sources.<name>.git.<repository>.remote.url`

The URL of the remote.

#### `sources.<name>.git.<repository>.remote.credential`

The name of the credential to use for the remote.

#### `sources.<name>.git.<repository>.proposals`

The configuration for the proposals for the git repository.

#### `sources.<name>.git.<repository>.proposals.credential`

The name of the credential to use for the proposals.

#### sources.\<name\>.oci

The configuration for an OCI source.

#### `sources.<name>.oci.<repository>`

The configuration for an OCI repository.

#### `sources.<name>.oci.<repository>.reference`

The reference to the OCI repository.

**Example:** `ghcr.io/get-glu/example-app`

#### `sources.<name>.oci.<repository>.credential`

The name of the credential to use for the OCI repository.

### server

#### `server.port`

The port to listen on for the server.

#### `server.host`

The host to listen on for the server.

#### `server.protocol`

The protocol to use for the server.

Valid values are `http` and `https`.

#### `server.cert_file`

The path to the certificate file to use for HTTPS.

#### `server.key_file`

The path to the key file to use for HTTPS.
