package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/get-glu/glu"
	"github.com/get-glu/glu/pkg/builder"
	"github.com/get-glu/glu/pkg/core"
)

// MockResource represents our mock resource state
type MockResource struct{}

func (m *MockResource) Digest() (string, error) {
	return "mock-digest", nil
}

func NewMockResource() *MockResource {
	return &MockResource{}
}

// MockSource implements a simple source for our mock resources
type MockSource struct {
	typ string
}

func NewMockSource(typ string) *MockSource {
	return &MockSource{typ: typ}
}

func (m *MockSource) Metadata() core.Metadata {
	return core.Metadata{
		Name: m.typ,
	}
}

func (m *MockSource) View(_ context.Context, _, _ core.Metadata, r *MockResource) error {
	return nil
}

func run(ctx context.Context) error {
	return builder.New[*MockResource](
		glu.NewSystem(ctx, glu.Name("mycorp", glu.Label("team", "ecommerce"))),
	).BuildPipeline(glu.Name("checkout"), NewMockResource, func(b builder.PipelineBuilder[*MockResource]) error {
		_, err := b.Configuration()
		if err != nil {
			return err
		}
		// OCI phase
		ociSource := NewMockSource("oci")
		ociPhase, err := b.NewPhase(
			glu.Name("oci"),
			ociSource,
		)
		if err != nil {
			return err
		}

		// Staging phase
		stagingSource := NewMockSource("git")
		stagingPhase, err := b.NewPhase(
			glu.Name("staging",
				glu.Label("environment", "staging"),
				glu.Label("region", "us-east-1"),
			),
			stagingSource,
			core.PromotesFrom(ociPhase),
		)
		if err != nil {
			return err
		}

		// Production phases
		prodEastSource := NewMockSource("git")
		b.NewPhase(
			glu.Name("production-east-1",
				glu.Label("environment", "production"),
				glu.Label("region", "us-east-1"),
			),
			prodEastSource,
			core.PromotesFrom(stagingPhase),
		)

		prodWestSource := NewMockSource("git")
		b.NewPhase(
			glu.Name("production-west-1",
				glu.Label("environment", "production"),
				glu.Label("region", "us-west-1"),
			),
			prodWestSource,
			core.PromotesFrom(stagingPhase),
		)

		return nil
	}).BuildPipeline(glu.Name("billing"), NewMockResource, func(b builder.PipelineBuilder[*MockResource]) error {
		// OCI phase
		fdOciSource := NewMockSource("oci")
		fdOciPhase, err := b.NewPhase(
			glu.Name("oci"),
			fdOciSource,
		)
		if err != nil {
			return err
		}

		// Staging phase
		fdStagingSource := NewMockSource("git")
		fdStagingPhase, err := b.NewPhase(
			glu.Name("staging",
				glu.Label("environment", "staging"),
				glu.Label("domain", "http://stage.billing.mycorp.com"),
			),
			fdStagingSource,
			core.PromotesFrom(fdOciPhase),
		)
		if err != nil {
			return err
		}

		// Production phase
		fdProdSource := NewMockSource("git")
		b.NewPhase(
			glu.Name("production",
				glu.Label("environment", "production"),
				glu.Label("domain", "https://prod.billing.mycorp.com"),
				glu.Label("ssl", "enabled"),
			),
			fdProdSource,
			core.PromotesFrom(fdStagingPhase),
		)

		return nil
	}).Run()
}

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("error running mock server", "error", err)
		os.Exit(1)
	}
}
