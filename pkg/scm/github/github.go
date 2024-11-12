package github

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"strings"

	"github.com/get-glu/glu/pkg/src/git"
	"github.com/google/go-github/v64/github"
)

const (
	GitHubPRNumberField = "github.pr.number"
)

var _ git.Proposer = (*SCM)(nil)

type SCM struct {
	client    *github.Client
	repoOwner string
	repoName  string
}

func New(client *github.Client, repoOwner, repoName string) *SCM {
	return &SCM{client: client, repoOwner: repoOwner, repoName: repoName}
}

func (s *SCM) GetCurrentProposal(ctx context.Context, baseBranch, branchPrefix string) (*git.Proposal, error) {
	var (
		prs      = s.listPRs(ctx, baseBranch)
		proposal *git.Proposal
	)

	for pr := range prs.All() {
		parts := strings.Split(pr.Head.GetRef(), "/")
		if strings.HasPrefix(pr.Head.GetRef(), branchPrefix) {
			proposal = &git.Proposal{
				BaseRevision: pr.Base.GetSHA(),
				BaseBranch:   pr.Base.GetRef(),
				Branch:       pr.Head.GetRef(),
				Digest:       parts[len(parts)-1],
				ExternalMetadata: map[string]any{
					GitHubPRNumberField: pr.GetNumber(),
				},
			}
			break
		}
	}

	if err := prs.Err(); err != nil {
		return nil, err
	}

	if proposal == nil {
		return nil, fmt.Errorf("base %q: prefix %q: %w", baseBranch, branchPrefix, git.ErrProposalNotFound)
	}

	return proposal, nil
}

func (s *SCM) CreateProposal(ctx context.Context, proposal *git.Proposal, opts git.ProposalOption) error {
	pr, _, err := s.client.PullRequests.Create(ctx, s.repoOwner, s.repoName, &github.NewPullRequest{
		Base:  github.String(proposal.BaseBranch),
		Head:  github.String(proposal.Branch),
		Title: github.String(proposal.Title),
		Body:  github.String(proposal.Body),
	})
	if err != nil {
		return err
	}

	slog.Info("proposal created", "scm_type", "github", "proposal_url", pr.GetHTMLURL())

	proposal.ExternalMetadata = map[string]any{
		GitHubPRNumberField: pr.GetNumber(),
	}

	if len(opts.Labels) > 0 {
		if _, _, err := s.client.Issues.AddLabelsToIssue(ctx, s.repoOwner, s.repoName, pr.GetNumber(), opts.Labels); err != nil {
			return err
		}
	}

	return nil
}

func (s *SCM) MergeProposal(ctx context.Context, proposal *git.Proposal) error {
	number, ok := proposal.ExternalMetadata[GitHubPRNumberField].(int)
	if !ok {
		slog.Warn("could not close pr", "reason", "missing PR number on proposal")
		return nil
	}

	_, _, err := s.client.PullRequests.Merge(ctx, s.repoOwner, s.repoName, number, "", &github.PullRequestOptions{
		MergeMethod: "merge",
	})

	return err
}

func (s *SCM) CloseProposal(ctx context.Context, proposal *git.Proposal) error {
	number, ok := proposal.ExternalMetadata[GitHubPRNumberField].(int)
	if !ok {
		slog.Warn("could not close pr", "reason", "missing PR number on proposal")
		return nil
	}

	_, _, err := s.client.PullRequests.Edit(ctx, s.repoOwner, s.repoName, number, &github.PullRequest{
		State: github.String("closed"),
	})

	return err
}

type prs struct {
	ctx       context.Context
	client    *github.PullRequestsService
	repoOwner string
	repoName  string
	base      string

	err error
}

func (s *SCM) listPRs(ctx context.Context, base string) *prs {
	return &prs{ctx, s.client.PullRequests, s.repoOwner, s.repoName, base, nil}
}

func (p *prs) Err() error {
	return p.err
}

func (p *prs) All() iter.Seq[*github.PullRequest] {
	return iter.Seq[*github.PullRequest](func(yield func(*github.PullRequest) bool) {
		opts := &github.PullRequestListOptions{
			Base: p.base,
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
			State: "open",
		}

		for {
			prs, resp, err := p.client.List(p.ctx, p.repoOwner, p.repoName, opts)
			if err != nil {
				p.err = err
				return
			}

			for _, pr := range prs {
				if !strings.HasPrefix(pr.Head.GetRef(), "glu/") {
					continue
				}

				if !yield(pr) {
					return
				}
			}

			if resp.NextPage == 0 {
				return
			}

			opts.Page = resp.NextPage
		}
	})
}
