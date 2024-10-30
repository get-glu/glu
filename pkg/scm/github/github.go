package github

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"strings"

	"github.com/flipt-io/glu"
	"github.com/google/go-github/v64/github"
)

const (
	GitHubPRNumberField = "github.pr.number"
)

type SCM struct {
	repoOwner string
	repoName  string
	client    *github.PullRequestsService
}

func (s *SCM) GetCurrentProposal(ctx context.Context, phase *glu.Phase, metadata *glu.Metadata) (*glu.Proposal, error) {
	var (
		prs      = s.listPRs(ctx, phase.Branch())
		proposal *glu.Proposal
	)

	branchPrefix := fmt.Sprintf("glu/%s/%s", phase.Name(), metadata.Name)

	for pr := range prs.All() {
		parts := strings.Split(pr.Head.GetRef(), "/")
		if strings.HasPrefix(pr.Head.GetRef(), branchPrefix) {
			proposal = &glu.Proposal{
				BaseRevision: parts[len(parts)-1],
				BaseBranch:   pr.Base.GetRef(),
				Branch:       pr.Head.GetRef(),
			}
			break
		}
	}

	if err := prs.Err(); err != nil {
		return nil, err
	}

	if proposal == nil {
		return nil, fmt.Errorf("phase %q app %q: %w", phase.Name, metadata.Name, glu.ErrProposalNotFound)
	}

	return proposal, nil
}

func (s *SCM) CreateProposal(ctx context.Context, proposal *glu.Proposal) error {
	pr, _, err := s.client.Create(ctx, s.repoOwner, s.repoName, &github.NewPullRequest{
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

	return nil
}

func (s *SCM) CloseProposal(ctx context.Context, proposal *glu.Proposal) error {
	number, ok := proposal.ExternalMetadata[GitHubPRNumberField].(int)
	if !ok {
		slog.Warn("could not close pr", "reason", "missing PR number on proposal")
		return nil
	}

	_, _, err := s.client.Edit(ctx, s.repoOwner, s.repoName, number, &github.PullRequest{
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
	return &prs{ctx, s.client, s.repoOwner, s.repoName, base, nil}
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
			State: "all",
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
