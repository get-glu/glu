package core

import (
	"context"

	"github.com/flipt-io/glu/pkg/fs"
)

type Proposal struct {
	BaseRevision string
	BaseBranch   string
	Branch       string
	Title        string
	Body         string

	ExternalMetadata map[string]any
}

type Repository interface {
	View(context.Context, *Phase, func(fs.Filesystem) error) error
	Update(context.Context, *Phase, *Metadata, func(fs.Filesystem) (string, error)) error
}

type Pipeline struct {
	ctx  context.Context
	name string
}

func NewPipeline(ctx context.Context, name string) *Pipeline {
	return &Pipeline{
		ctx:  ctx,
		name: name,
	}
}

func (p *Pipeline) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

type Phase struct {
	pipeline *Pipeline
	name     string
	// TODO(georgemac): make optionally configurable
	branch string
	repo   Repository
}

func (p *Phase) Name() string {
	return p.name
}

func (p *Phase) Branch() string {
	return p.branch
}

func (p *Phase) Repository() Repository {
	return p.repo
}

func (p *Pipeline) NewPhase(name string, repo Repository) *Phase {
	return &Phase{
		pipeline: p,
		name:     name,
		branch:   "main",
		repo:     repo,
	}
}

type Metadata struct {
	Name   string
	Labels map[string]string
}

type App interface {
	ReadFrom(context.Context, *Phase, fs.Filesystem) error
	WriteTo(_ context.Context, _ *Phase, _ fs.Filesystem) error
}
