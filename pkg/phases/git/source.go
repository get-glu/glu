package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"path"
	"sync"

	"github.com/get-glu/glu/internal/git"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/go-git/go-git/v5/plumbing"
)

const (
	AnnotationGitBaseRefKey     = "dev.getglu.git.base_ref"
	AnnotationGitHeadSHAKey     = "dev.getglu.git.head_sha"
	AnnotationProposalNumberKey = "dev.getglu.git.proposal.number"
	AnnotationProposalURLKey    = "dev.getglu.git.proposal.url"
)

var (
	// ErrProposalNotFound is returned when a proposal cannot be located
	ErrProposalNotFound = errors.New("proposal not found")
)

type Resource interface {
	core.Resource
	ReadFrom(context.Context, core.Descriptor, fs.Filesystem) error
	WriteTo(context.Context, core.Descriptor, fs.Filesystem) error
}

type Proposer interface {
	GetCurrentProposal(_ context.Context, baseBranch, branchPrefix string) (*Proposal, error)
	CreateProposal(context.Context, *Proposal, ProposalOption) error
	MergeProposal(context.Context, *Proposal) error
	CloseProposal(context.Context, *Proposal) error
}

// Proposal contains the fields necessary to propose a resource update
// to a Repository.
type Proposal struct {
	BaseRevision string
	BaseBranch   string
	Branch       string
	HeadRevision string
	Digest       string
	Title        string
	Body         string

	Annotations map[string]string
}

type RefLog[R core.Resource] interface {
	CreateReference(ctx context.Context, phase core.Descriptor) error
	RecordLatest(ctx context.Context, phase core.Descriptor, resource R, annotations map[string]string) error
	History(ctx context.Context, phase core.Descriptor) ([]core.State, error)
}

type Phase[R Resource] struct {
	mu              sync.RWMutex
	pipeline        string
	meta            core.Metadata
	newFn           func() R
	repo            *git.Repository
	proposer        Proposer
	proposeChange   bool
	proposalOptions ProposalOption
	log             RefLog[R]
}

func (p *Phase[A]) Descriptor() core.Descriptor {
	return core.Descriptor{
		Kind:     "git",
		Pipeline: p.pipeline,
		Metadata: p.meta,
	}
}

type ProposalOption struct {
	Labels []string
}

// ProposeChanges configures the phase to propose the change (via PR or MR)
// as opposed to directly integrating it into the target trunk branch.
func ProposeChanges[A Resource](opts ProposalOption) containers.Option[Phase[A]] {
	return func(i *Phase[A]) {
		i.proposeChange = true
		i.proposalOptions = opts
	}
}

func NewPhase[R Resource](
	pipeline string,
	meta core.Metadata,
	newFn func() R,
	repo *git.Repository,
	proposer Proposer,
	opts ...containers.Option[Phase[R]],
) (_ *Phase[R]) {
	phase := &Phase[R]{
		pipeline: pipeline,
		meta:     meta,
		newFn:    newFn,
		repo:     repo,
		proposer: proposer,
	}

	containers.ApplyAll(phase, opts...)

	if phase.log != nil {
		phase.repo.Subscribe(phase)
	}

	return phase
}

type Branched interface {
	Branch(phase core.Descriptor) string
}

func (g *Phase[R]) Get(ctx context.Context) (core.Resource, error) {
	return g.GetResource(ctx)
}

func (g *Phase[R]) GetResource(ctx context.Context) (R, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := g.newFn()
	desc := g.Descriptor()

	opts := []containers.Option[git.ViewUpdateOptions]{}
	if branched, ok := core.Resource(r).(Branched); ok {
		opts = append(opts, git.WithBranch(branched.Branch(desc)))
	}

	if err := g.repo.View(ctx, func(hash plumbing.Hash, fs fs.Filesystem) error {
		return r.ReadFrom(ctx, desc, fs)
	}, opts...); err != nil {
		return r, err
	}

	return r, nil
}

func (g *Phase[R]) History(ctx context.Context) ([]core.State, error) {
	if g.log == nil {
		return nil, nil
	}

	return g.log.History(ctx, g.Descriptor())
}

func (p *Phase[R]) branch() string {
	if branched, ok := core.Resource(p.newFn()).(Branched); ok {
		return branched.Branch(p.Descriptor())
	}
	return p.repo.DefaultBranch()
}

func (p *Phase[R]) Branches() []string {
	return []string{p.branch()}
}

func (p *Phase[R]) Notify(ctx context.Context, refs map[string]string) error {
	ref, ok := refs[p.branch()]
	if !ok {
		slog.Debug("reference not found on notify", "branch", p.branch(), "refs", refs)
		return nil
	}

	r := p.newFn()
	if err := p.repo.View(ctx, func(hash plumbing.Hash, fs fs.Filesystem) error {
		return r.ReadFrom(ctx, p.Descriptor(), fs)
	}, git.WithRevision(plumbing.NewHash(ref))); err != nil {
		return err
	}

	// record latest
	p.log.RecordLatest(ctx, p.Descriptor(), r, map[string]string{
		AnnotationGitHeadSHAKey: ref,
	})

	return nil
}

type commitMessage[R Resource] interface {
	// CommitMessage is an optional git specific method for overriding generated commit messages.
	// The function is provided with the source phases metadata and the previous value of resource.
	CommitMessage(phase core.Descriptor, from R) (string, error)
}

type proposalTitle[R Resource] interface {
	// ProposalTitle is an optional git specific method for overriding generated proposal message (PR/MR) title message.
	// The function is provided with the source phases metadata and the previous value of resource.
	ProposalTitle(phase core.Descriptor, from R) (string, error)
}

type proposalBody[R Resource] interface {
	// ProposalBody is an optional git specific method for overriding generated proposal body (PR/MR) body message.
	// The function is provided with the source phases metadata and the previous value of resource.
	ProposalBody(phase core.Descriptor, from R) (string, error)
}

func (p *Phase[R]) Update(ctx context.Context, from, to R) (map[string]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// perform an initial fetch to ensure we're up to date
	// TODO(georgmac): scope to phase branch and proposal prefix
	err := p.repo.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching upstream during update: %w", err)
	}

	// use the target resources branch if it implementes an override
	desc := p.Descriptor()
	baseBranch := p.repo.DefaultBranch()
	if branched, ok := core.Resource(to).(Branched); ok {
		baseBranch = branched.Branch(desc)
	}

	annotations := map[string]string{
		AnnotationGitBaseRefKey: baseBranch,
	}

	if !p.proposeChange {
		annotations[AnnotationGitHeadSHAKey], err = p.updateAndPush(ctx, from, to, git.WithBranch(baseBranch))
		if err != nil {
			return nil, err
		}

		return annotations, nil
	}

	if p.proposer == nil {
		return nil, errors.New("proposal requested but not configured")
	}

	return p.propose(ctx, from, to, baseBranch)
}

func (p *Phase[R]) propose(ctx context.Context, from, to R, baseBranch string) (map[string]string, error) {
	slog := slog.With("name", p.meta.Name)
	desc := p.Descriptor()

	baseRev, err := p.repo.Resolve(baseBranch)
	if err != nil {
		return nil, fmt.Errorf("resolving base branch %q: %w", baseBranch, err)
	}

	digest, err := to.Digest()
	if err != nil {
		return nil, err
	}

	// create branch name and check if this phase, resource and state has previously been observed
	var (
		branchPrefix = fmt.Sprintf("glu/%s/%s", p.pipeline, p.meta.Name)
		branch       = path.Join(branchPrefix, digest)
	)

	// ensure branch exists locally either way
	if err := p.repo.CreateBranchIfNotExists(branch, git.WithBase(baseBranch)); err != nil {
		return nil, err
	}

	proposal, err := p.proposer.GetCurrentProposal(ctx, baseBranch, branchPrefix)
	if err != nil {
		if !errors.Is(err, ErrProposalNotFound) {
			return nil, err
		}

		slog.Debug("proposal not found")
	}

	options := []containers.Option[git.ViewUpdateOptions]{git.WithBranch(branch)}
	if proposal != nil {
		// there is an existing proposal
		slog.Debug("proposal found", "base", proposal.BaseBranch, "base_revision", proposal.BaseRevision)
		if proposal.BaseRevision != baseRev.String() {
			// we're potentially going to force update the branch to move the base
			options = append(options, git.WithForce)
		} else if proposal.Digest == digest {
			// nothing has changed since the last promotion and proposals
			slog.Debug("skipping proposal", "reason", "AlreadyExistsAndUpToDate")

			return annotations(proposal), nil
		}

		head, err := p.updateAndPush(ctx, from, to, options...)
		if err != nil {
			return nil, fmt.Errorf("updating existing proposal: %w", err)
		}

		// we're updating the head position of an existing proposal
		// so we need to update the value of head in the returned annotations
		annotations := annotations(proposal)
		annotations[AnnotationGitHeadSHAKey] = head

		return annotations, nil
	}

	if head, err := p.updateAndPush(ctx, from, to, options...); err != nil {
		if !errors.Is(err, core.ErrNoChange) {
			return nil, err
		}

		// we check here to see if the update produced no changes due to either
		// the branch previously existing and had been advanced already or
		// the branch was just created and matches the base after update
		// if it is the latter then our update produces no change
		// otherwise, we just need to open the pull request again and continue
		if baseRev.String() == head {
			return nil, core.ErrNoChange
		}
	}

	fromDigest, err := from.Digest()
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf("Update %s", p.meta.Name)
	if p, ok := core.Resource(to).(proposalTitle[R]); ok {
		title, err = p.ProposalTitle(desc, from)
		if err != nil {
			return nil, err
		}
	}

	body := fmt.Sprintf(`| from | to |
| -- | -- |
| %s | %s |
`, fromDigest, digest)
	if b, ok := core.Resource(to).(proposalBody[R]); ok {
		body, err = b.ProposalBody(desc, from)
		if err != nil {
			return nil, err
		}
	}

	proposal = &Proposal{
		BaseRevision: baseRev.String(),
		BaseBranch:   baseBranch,
		Branch:       branch,
		Title:        title,
		Body:         body,
	}

	if err := p.proposer.CreateProposal(ctx, proposal, p.proposalOptions); err != nil {
		return nil, err
	}

	return annotations(proposal), nil
}

func (p *Phase[R]) updateAndPush(ctx context.Context, from, to R, opts ...containers.Option[git.ViewUpdateOptions]) (string, error) {
	slog := slog.With("name", p.meta.Name)

	desc := p.Descriptor()
	update := func(fs fs.Filesystem) (message string, err error) {
		message = fmt.Sprintf("Update %s", p.meta.Name)
		if m, ok := core.Resource(to).(commitMessage[R]); ok {
			message, err = m.CommitMessage(desc, from)
			if err != nil {
				return "", fmt.Errorf("overriding commit message during update: %w", err)
			}
		}

		if err := to.WriteTo(ctx, desc, fs); err != nil {
			return "", err
		}

		return message, nil
	}

	head, err := p.repo.UpdateAndPush(ctx, update, opts...)
	if err != nil {
		if errors.Is(err, git.ErrEmptyCommit) {
			slog.Debug("promotion produced no changes")

			return "", core.ErrNoChange
		}

		return "", err
	}

	return head.String(), nil
}

func annotations(proposal *Proposal) map[string]string {
	a := map[string]string{
		AnnotationGitBaseRefKey: proposal.BaseBranch,
		AnnotationGitHeadSHAKey: proposal.HeadRevision,
	}

	maps.Insert(a, maps.All(proposal.Annotations))
	return a
}
