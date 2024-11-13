package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/get-glu/glu"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/phases"
)

// MockResource represents our mock resource state
type MockResource struct {
}

func (m *MockResource) Digest() (string, error) {
	return "mock-digest", nil
}

func NewMockResource() *MockResource {
	return &MockResource{}
}

// MockSource implements a simple source for our mock resources
type MockSource struct {
}

func NewMockSource() *MockSource {
	return &MockSource{}
}

func (m *MockSource) View(_ context.Context, _, _ core.Metadata, r *MockResource) error {
	return nil
}

func run(ctx context.Context) error {
	system := glu.NewSystem(ctx)
	system.AddPipeline(func(ctx context.Context, config *glu.Config) (glu.Pipeline, error) {
		// Create cloud-controller pipeline
		ccPipeline := glu.NewPipeline(glu.Name("cloud-controller"), NewMockResource)

		// OCI phase
		ociSource := NewMockSource()
		ociPhase, err := phases.New(
			glu.Name("cloud-controller-oci", glu.Label("type", "oci"), glu.Label("status", "running"), glu.Label("version", "v1.2.3")),
			ccPipeline,
			ociSource,
		)
		if err != nil {
			return nil, err
		}

		// Staging phase
		stagingSource := NewMockSource()
		stagingPhase, err := phases.New(
			glu.Name("cloud-controller-staging",
				glu.Label("type", "staging"),
				glu.Label("environment", "staging"),
				glu.Label("region", "us-east-1"),
			),
			ccPipeline,
			stagingSource,
			core.PromotesFrom(ociPhase),
		)
		if err != nil {
			return nil, err
		}

		// Production phases
		prodEastSource := NewMockSource()
		phases.New(
			glu.Name("cloud-controller-production-east-1",
				glu.Label("type", "production"),
				glu.Label("environment", "production"),
				glu.Label("region", "us-east-1"),
				glu.Label("replicas", "3"),
			),
			ccPipeline,
			prodEastSource,
			core.PromotesFrom(stagingPhase),
		)

		prodWestSource := NewMockSource()
		phases.New(
			glu.Name("cloud-controller-production-west-1",
				glu.Label("type", "production"),
				glu.Label("environment", "production"),
				glu.Label("region", "us-west-1"),
				glu.Label("replicas", "3"),
			),
			ccPipeline,
			prodWestSource,
			core.PromotesFrom(stagingPhase),
		)

		return ccPipeline, nil
	})

	system.AddPipeline(func(ctx context.Context, config *glu.Config) (glu.Pipeline, error) {
		// Create frontdoor pipeline
		fdPipeline := glu.NewPipeline(glu.Name("frontdoor"), NewMockResource)

		// OCI phase
		fdOciSource := NewMockSource()
		fdOciPhase, err := phases.New(
			glu.Name("frontdoor-oci", glu.Label("type", "oci"), glu.Label("version", "v2.0.1"), glu.Label("builder", "docker")),
			fdPipeline,
			fdOciSource,
		)
		if err != nil {
			return nil, err
		}

		// Staging phase
		fdStagingSource := NewMockSource()
		fdStagingPhase, err := phases.New(
			glu.Name("frontdoor-staging",
				glu.Label("type", "staging"),
				glu.Label("environment", "staging"),
				glu.Label("domain", "stage.example.com"),
			),
			fdPipeline,
			fdStagingSource,
			core.PromotesFrom(fdOciPhase),
		)
		if err != nil {
			return nil, err
		}

		// Production phase
		fdProdSource := NewMockSource()
		phases.New(
			glu.Name("frontdoor-production",
				glu.Label("type", "production"),
				glu.Label("environment", "production"),
				glu.Label("domain", "prod.example.com"),
				glu.Label("ssl", "enabled"),
			),
			fdPipeline,
			fdProdSource,
			core.PromotesFrom(fdStagingPhase),
		)

		return fdPipeline, nil
	})

	return system.Run()
}

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("error running mock server", "error", err)
		os.Exit(1)
	}
}
