package core

import (
	"context"

	"github.com/flipt-io/glu/pkg/fs"
)

type Proposal struct {
	BaseRevision string
	BaseBranch   string
	Branch       string
	Digest       string
	Title        string
	Body         string

	ExternalMetadata map[string]any
}

type Repository interface {
	View(context.Context, Resource, func(fs.Filesystem) error) error
	Update(_ context.Context, from, to Resource, _ func(fs.Filesystem) (string, error)) error
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

type Metadata struct {
	Name   string
	Phase  string
	Labels map[string]string
}

type Resource interface {
	Metadata() *Metadata
	Digest() (string, error)
	ReadFrom(context.Context, fs.Filesystem) error
	WriteTo(context.Context, fs.Filesystem) error
}
