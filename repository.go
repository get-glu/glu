package glu

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/flipt-io/glu/pkg/containers"
	"github.com/flipt-io/glu/pkg/fs"
	"github.com/flipt-io/glu/pkg/git"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

type Repository interface {
	View(context.Context, func(fs.Filesystem) error) error
	Update(_ context.Context, branch string, _ func(fs.Filesystem) (string, error)) error
}

type GitRepository struct {
	name   string
	source *git.Source
}

func NewGitRepository(name string) (*GitRepository, error) {
	method, err := ssh.DefaultAuthBuilder("git")
	if err != nil {
		return nil, err
	}

	opts := []containers.Option[git.Source]{git.WithAuth(method)}

	if remote := getEnvVar(name, "REMOTE_URL"); remote != "" {
		slog.Debug("configuring remote", "remote", remote)

		opts = append(opts, git.WithRemote("origin", remote))
	}

	source, err := git.NewSource(context.Background(), slog.Default(), opts...)
	if err != nil {
		return nil, err
	}

	return &GitRepository{source: source}, nil
}

func getEnvVar(name, k string) string {
	return os.Getenv(strings.ToUpper(fmt.Sprintf("GLU_REPOSITORY_%s_%s", name, k)))
}

func (g *GitRepository) View(ctx context.Context, fn func(fs.Filesystem) error) error {
	return g.source.View(ctx, "main", func(hash plumbing.Hash, fs fs.Filesystem) error {
		return fn(fs)
	})
}

func (g *GitRepository) Update(ctx context.Context, branch string, fn func(fs.Filesystem) (string, error)) error {
	if err := g.source.CreateBranchIfNotExists(branch); err != nil {
		return err
	}

	if _, err := g.source.UpdateAndPush(ctx, branch, nil, func(fs fs.Filesystem) (string, error) {
		return fn(fs)
	}); err != nil {
		return err
	}

	return nil
}
