package git

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage"
	gitfilesystem "github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
)

type Repository struct {
	logger          *slog.Logger
	remote          *config.RemoteConfig
	defaultBranch   string
	auth            transport.AuthMethod
	insecureSkipTLS bool
	caBundle        []byte
	localPath       string
	readme          []byte
	sigName         string
	sigEmail        string
	maxOpenDescs    int

	mu   sync.RWMutex
	repo *git.Repository

	subs []Subscriber

	pollInterval time.Duration
	cancel       func()
	done         chan struct{}
}

type Subscriber interface {
	Branches() []string
	Notify(ctx context.Context, refs map[string]string) error
}

func NewRepository(ctx context.Context, logger *slog.Logger, opts ...containers.Option[Repository]) (*Repository, error) {
	repo, empty, err := newRepository(ctx, logger, opts...)
	if err != nil {
		return nil, err
	}

	if empty {
		logger.Warn("repository empty, attempting to add and push a README")
		// add initial readme if repo is empty
		if _, err := repo.UpdateAndPush(ctx, func(fs fs.Filesystem) (string, error) {
			fi, err := fs.OpenFile("README.md", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
			if err != nil {
				return "", err
			}

			if _, err := fi.Write(repo.readme); err != nil {
				return "", err
			}

			if err := fi.Close(); err != nil {
				return "", err
			}

			return "add initial README", nil
		}); err != nil {
			return nil, err
		}
	}

	repo.startPolling(ctx)

	return repo, nil
}

// newRepository is a wrapper around the core *git.Repository
// It handles configuring a repository source appropriately based on our configuration
// It also exposes some common operations and ensures safe concurrent access while fetching and pushing
func newRepository(ctx context.Context, logger *slog.Logger, opts ...containers.Option[Repository]) (_ *Repository, empty bool, err error) {
	r := &Repository{
		logger:        logger,
		defaultBranch: "main",
		sigName:       "glu bot",
		sigEmail:      "bot@get-glu.dev",
		readme:        []byte(`# Glu Configuration Repository`),
		// we initialize with a noop function incase
		// we dont start the polling loop
		cancel: func() {},
		done:   make(chan struct{}),
	}

	containers.ApplyAll(r, opts...)

	// we initially assume the repo is empty because we start
	// with an in-memory blank slate
	empty = true
	storage := (storage.Storer)(memory.NewStorage())
	r.repo, err = git.InitWithOptions(storage, nil, git.InitOptions{
		DefaultBranch: plumbing.NewBranchReferenceName(r.defaultBranch),
	})
	if err != nil {
		return nil, empty, err
	}

	if r.localPath != "" {
		storage = gitfilesystem.NewStorageWithOptions(osfs.New(r.localPath), cache.NewObjectLRUDefault(), gitfilesystem.Options{
			MaxOpenDescriptors: r.maxOpenDescs,
		})

		entries, err := os.ReadDir(r.localPath)
		if empty = err != nil || len(entries) == 0; empty {
			// either its empty or there was an error opening the file
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, empty, err
			}

			r.repo, err = git.InitWithOptions(storage, nil, git.InitOptions{
				DefaultBranch: plumbing.NewBranchReferenceName(r.defaultBranch),
			})
			if err != nil {
				return nil, empty, err
			}
		} else {
			// opened successfully and there is contents so we assume not empty
			r.repo, err = git.Open(storage, nil)
			if err != nil {
				return nil, empty, err
			}
		}
	}

	if r.remote != nil {
		if len(r.remote.URLs) == 0 {
			return nil, empty, errors.New("must supply at-least one remote URL")
		}

		if _, err = r.repo.CreateRemote(r.remote); err != nil {
			if !errors.Is(err, git.ErrRemoteExists) {
				return nil, empty, err
			}
		}

		// given an upstream has been configured we're going to start
		// by changing our assumption to the repository having contents
		empty = false

		// do an initial fetch to setup remote tracking branches
		if err := r.Fetch(ctx); err != nil {
			if !errors.Is(err, transport.ErrEmptyRemoteRepository) &&
				!errors.Is(err, git.NoMatchingRefSpecError{}) {
				return nil, empty, fmt.Errorf("performing initial fetch: %w", err)
			}

			// the remote was reachable but either its contents was completely empty
			// or our default branch doesn't exist and so we decide to seed it
			empty = true

			logger.Debug("initial fetch empty", slog.String("reference", r.defaultBranch), "error", err)
		}
	}

	if plumbing.IsHash(r.defaultBranch) {
		// if we still need to add an initial commit to the repository then we assume they couldn't
		// have predicted the initial hash and return reference not found
		if empty {
			return nil, empty, fmt.Errorf("target repository is empty: %w", plumbing.ErrReferenceNotFound)
		}

		return r, empty, r.repo.Storer.SetReference(plumbing.NewHashReference(plumbing.HEAD, plumbing.NewHash(r.defaultBranch)))
	}

	return r, empty, nil
}

func (r *Repository) startPolling(ctx context.Context) {
	if r.pollInterval == 0 {
		close(r.done)
		return
	}

	go func() {
		defer close(r.done)

		ticker := time.NewTicker(r.pollInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.Fetch(ctx); err != nil {
					r.logger.Error("error performing fetch", "error", err)
					continue
				}

				r.logger.Debug("fetch successful")
			}
		}
	}()
}

func (r *Repository) DefaultBranch() string {
	return r.defaultBranch
}

func (r *Repository) Close() error {
	r.cancel()

	<-r.done

	return nil
}

// Subscribe registers the functions for the given branch name.
// It will be called each time the branch is updated while holding a lock.
func (r *Repository) Subscribe(sub Subscriber) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.subs = append(r.subs, sub)
}

func (r *Repository) fetchHeads() []string {
	heads := map[string]struct{}{r.defaultBranch: {}}
	for _, sub := range r.subs {
		for _, head := range sub.Branches() {
			heads[head] = struct{}{}
		}
	}

	return slices.Collect(maps.Keys(heads))
}

// Fetch does a fetch for the requested head names on a configured remote.
// If the remote is not defined, then it is a silent noop.
// Iff specific is explicitly requested then only the heads in specific are fetched.
// Otherwise, it fetches all previously tracked head references.
func (r *Repository) Fetch(ctx context.Context, specific ...string) (err error) {
	if r.remote == nil {
		return nil
	}

	updatedRefs := map[string]plumbing.Hash{}
	r.mu.Lock()
	defer func() {
		r.mu.Unlock()

		// update subscribers if any matching and requested references
		// are updated while processing this fetch
		if len(updatedRefs) > 0 {
			r.updateSubs(ctx, updatedRefs)
		}
	}()

	heads := specific
	if len(heads) == 0 {
		heads = r.fetchHeads()
	}

	var refSpecs = []config.RefSpec{}

	for _, head := range heads {
		refSpec := config.RefSpec(
			fmt.Sprintf("+%s:%s",
				plumbing.NewBranchReferenceName(head),
				plumbing.NewRemoteReferenceName(r.remote.Name, head),
			),
		)

		r.logger.Debug("preparing refspec for fetch", slog.String("refspec", refSpec.String()))

		refSpecs = append(refSpecs, refSpec)
	}

	if err := r.repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName:      r.remote.Name,
		Auth:            r.auth,
		CABundle:        r.caBundle,
		InsecureSkipTLS: r.insecureSkipTLS,
		RefSpecs:        refSpecs,
	}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}

	allRefs, err := r.repo.References()
	if err != nil {
		return err
	}

	if err := allRefs.ForEach(func(ref *plumbing.Reference) error {
		// we're only interested in updates to remotes
		if !ref.Name().IsRemote() {
			return nil
		}

		for _, head := range heads {
			name := strings.TrimPrefix(ref.Name().String(), "refs/remotes/origin/")
			if refMatch(name, head) {
				updatedRefs[name] = ref.Hash()
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (r *Repository) ListCommits(ctx context.Context, branch, from string, filter func(string) bool) (_ iter.Seq[*object.Commit], err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.logger.Debug("ListCommits", slog.String("branch", branch))

	hash := plumbing.NewHash(from)
	if from == "" || !plumbing.IsHash(from) {
		hash, err = r.Resolve(branch)
		if err != nil {
			return nil, err
		}
	}

	i, err := r.repo.Log(&git.LogOptions{
		From:       hash,
		Order:      git.LogOrderBSF,
		PathFilter: filter,
	})
	if err != nil {
		return nil, err
	}

	return iter.Seq[*object.Commit](func(yield func(*object.Commit) bool) {
		if err := i.ForEach(func(c *object.Commit) error {
			if !yield(c) {
				return storer.ErrStop
			}

			return nil
		}); err != nil {
			return
		}
	}), nil
}

type ViewUpdateOptions struct {
	branch string
	// revision on View predicates it to the specific hash
	// revision on Update returns a conflict error if the branch head does not match
	revision plumbing.Hash
	// force configures an update to ignore any conflicts when attempting to push
	force bool
}

func (r *Repository) getOptions(opts ...containers.Option[ViewUpdateOptions]) *ViewUpdateOptions {
	defaultOptions := &ViewUpdateOptions{branch: r.defaultBranch}
	containers.ApplyAll(defaultOptions, opts...)
	return defaultOptions
}

func WithBranch(branch string) containers.Option[ViewUpdateOptions] {
	return func(vuo *ViewUpdateOptions) {
		vuo.branch = branch
	}
}

func WithRevision(rev plumbing.Hash) containers.Option[ViewUpdateOptions] {
	return func(vuo *ViewUpdateOptions) {
		vuo.revision = rev
	}
}

func WithForce(vuo *ViewUpdateOptions) {
	vuo.force = true
}

func (r *Repository) View(ctx context.Context, fn func(hash plumbing.Hash, fs fs.Filesystem) error, opts ...containers.Option[ViewUpdateOptions]) (err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	options := r.getOptions(opts...)

	hash := options.revision
	if hash == plumbing.ZeroHash {
		hash, err = r.Resolve(options.branch)
		if err != nil {
			return err
		}
	}

	r.logger.Debug("View", slog.String("branch", options.branch), slog.String("revision", hash.String()))

	fs, err := r.newFilesystem(hash)
	if err != nil {
		return err
	}

	return fn(hash, fs)
}

func (r *Repository) UpdateAndPush(ctx context.Context, fn func(fs fs.Filesystem) (string, error), opts ...containers.Option[ViewUpdateOptions]) (hash plumbing.Hash, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var (
		options = r.getOptions(opts...)
		branch  = options.branch
		rev     = options.revision
	)

	hash, err = r.Resolve(branch)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	if rev != plumbing.ZeroHash && rev != hash {
		return hash, fmt.Errorf("base revision %q has changed (now %q): %w", rev, hash, errors.New("conflict"))
	}

	// if rev == nil then hash will be the zero hash
	fs, err := r.newFilesystem(hash)
	if err != nil {
		return hash, err
	}

	msg, err := fn(fs)
	if err != nil {
		return hash, err
	}

	commit, err := fs.commit(ctx, msg)
	if err != nil {
		return hash, err
	}

	if r.remote != nil {
		local := plumbing.NewBranchReferenceName(branch)
		if err := r.repo.Storer.SetReference(plumbing.NewHashReference(local, commit.Hash)); err != nil {
			return hash, err
		}

		spec := fmt.Sprintf("%[1]s:%[1]s", local)
		if options.force {
			spec = "+" + spec
		}

		if err := r.repo.PushContext(ctx, &git.PushOptions{
			RemoteName:      r.remote.Name,
			Auth:            r.auth,
			CABundle:        r.caBundle,
			InsecureSkipTLS: r.insecureSkipTLS,
			RefSpecs: []config.RefSpec{
				config.RefSpec(spec),
			},
		}); err != nil {
			return hash, err
		}
	}

	remoteName := "origin"
	if r.remote != nil {
		remoteName = r.remote.Name
	}

	// update remote tracking reference to match
	remoteRef := plumbing.NewHashReference(
		plumbing.NewRemoteReferenceName(remoteName, branch),
		commit.Hash)

	if err := r.repo.Storer.SetReference(remoteRef); err != nil {
		return hash, err
	}

	// update references
	r.updateSubs(ctx, map[string]plumbing.Hash{branch: commit.Hash})

	return commit.Hash, nil
}

func (r *Repository) updateSubs(ctx context.Context, refs map[string]plumbing.Hash) {
	// update subscribers for each matching ref
	for _, sub := range r.subs {
		matched := map[string]string{}
		for ref, hash := range refs {
			for _, branch := range sub.Branches() {
				if refMatch(ref, branch) {
					matched[ref] = hash.String()
				}
			}
		}

		if err := sub.Notify(ctx, matched); err != nil {
			r.logger.Error("while updating subscriber", "error", err)
		}
	}
}

func refMatch(ref, pattern string) bool {
	if !strings.Contains(pattern, "*") {
		return ref == pattern
	}

	return strings.HasPrefix(ref, pattern[:strings.Index(pattern, "*")])
}

func (r *Repository) Resolve(branch string) (plumbing.Hash, error) {
	reference, err := r.repo.Reference(plumbing.NewRemoteReferenceName("origin", branch), true)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	return reference.Hash(), nil
}

type CreateBranchOptions struct {
	base string
}

func WithBase(name string) containers.Option[CreateBranchOptions] {
	return func(cbo *CreateBranchOptions) {
		cbo.base = name
	}
}

func (r *Repository) CreateBranchIfNotExists(branch string, opts ...containers.Option[CreateBranchOptions]) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	remoteName := "origin"
	if r.remote != nil {
		remoteName = r.remote.Name
	}

	remoteRef := plumbing.NewRemoteReferenceName(remoteName, branch)
	if _, err := r.repo.Reference(remoteRef, true); err == nil {
		// reference already exists
		return nil
	}

	opt := CreateBranchOptions{base: r.defaultBranch}

	containers.ApplyAll(&opt, opts...)

	reference, err := r.repo.Reference(plumbing.NewRemoteReferenceName(remoteName, opt.base), true)
	if err != nil {
		return fmt.Errorf("base reference %q not found: %w", opt.base, err)
	}

	return r.repo.Storer.SetReference(plumbing.NewHashReference(remoteRef,
		reference.Hash()))
}

func (r *Repository) newFilesystem(hash plumbing.Hash) (_ *filesystem, err error) {
	var (
		commit *object.Commit
		tree   = &object.Tree{}
	)

	// zero hash assumes we're building from an emptry repository
	// the caller needs to validate whether this is true or not
	// before calling newFilesystem with zero hash
	if hash != plumbing.ZeroHash {
		commit, err = r.repo.CommitObject(hash)
		if err != nil {
			return nil, fmt.Errorf("getting branch commit: %w", err)
		}

		tree, err = commit.Tree()
		if err != nil {
			return nil, err
		}
	}

	return &filesystem{
		logger:   r.logger,
		base:     commit,
		tree:     tree,
		storage:  r.repo.Storer,
		sigName:  r.sigName,
		sigEmail: r.sigEmail,
	}, nil
}

func WithRemote(name, url string) containers.Option[Repository] {
	return func(r *Repository) {
		r.remote = &config.RemoteConfig{
			Name: "origin",
			URLs: []string{url},
		}
	}
}

// WithDefaultBranch configures the default branch used to initially seed
// the repo, or base other branches on when they're not already present
// in the upstream.
func WithDefaultBranch(ref string) containers.Option[Repository] {
	return func(s *Repository) {
		s.defaultBranch = ref
	}
}

// WithAuth returns an option which configures the auth method used
// by the provided source.
func WithAuth(auth transport.AuthMethod) containers.Option[Repository] {
	return func(s *Repository) {
		s.auth = auth
	}
}

// WithInsecureTLS returns an option which configures the insecure TLS
// setting for the provided source.
func WithInsecureTLS(insecureSkipTLS bool) containers.Option[Repository] {
	return func(s *Repository) {
		s.insecureSkipTLS = insecureSkipTLS
	}
}

// WithCABundle returns an option which configures the CA Bundle used for
// validating the TLS connection to the provided source.
func WithCABundle(caCertBytes []byte) containers.Option[Repository] {
	return func(s *Repository) {
		if caCertBytes != nil {
			s.caBundle = caCertBytes
		}
	}
}

// WithFilesystemStorage configures the Git repository to clone into
// the local filesystem, instead of the default which is in-memory.
// The provided path is location for the dotgit folder.
func WithFilesystemStorage(path string) containers.Option[Repository] {
	return func(r *Repository) {
		r.localPath = path
	}
}

// WithSignature sets the default signature name and email when the signature
// cannot be derived from the request context.
func WithSignature(name, email string) containers.Option[Repository] {
	return func(r *Repository) {
		r.sigName = name
		r.sigEmail = email
	}
}

// WithInterval sets the period between automatic fetches from the upstream (if a remote is configured)
func WithInterval(interval time.Duration) containers.Option[Repository] {
	return func(r *Repository) {
		r.pollInterval = interval
	}
}

// WithMaxOpenDescriptors sets the maximum number of open file descriptors when using filesystem backed storage
func WithMaxOpenDescriptors(n int) containers.Option[Repository] {
	return func(r *Repository) {
		r.maxOpenDescs = n
	}
}
