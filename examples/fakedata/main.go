package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/get-glu/glu"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/edges"
	"github.com/get-glu/glu/pkg/pipelines"
)

func run(ctx context.Context) error {
	system := glu.NewSystem(ctx, glu.Name("mycorp", glu.Label("team", "ecommerce")))
	config, err := system.Configuration()
	if err != nil {
		return err
	}

	checkout := pipelines.NewBuilder(config, glu.Name("checkout"), NewMockResource)
	// oci promotes to staging
	stagingPhase := checkout.
		// oci phase
		NewPhase(func(pipelines.Builder[*MockResource]) (edges.Phase[*MockResource], error) {
			return NewMockPhase("checkout", "oci", "oci"), nil
		}).
		// staging phase
		PromotesTo(func(pipelines.Builder[*MockResource]) (edges.UpdatablePhase[*MockResource], error) {
			return NewMockPhase("checkout", "git", "staging"), nil
		})

	// staging promotes to prod-east-1
	stagingPhase.PromotesTo(func(pipelines.Builder[*MockResource]) (edges.UpdatablePhase[*MockResource], error) {
		return NewMockPhase("checkout", "git", "production-east-1"), nil
	})

	// staging promotes to prod-west-1
	stagingPhase.PromotesTo(func(pipelines.Builder[*MockResource]) (edges.UpdatablePhase[*MockResource], error) {
		return NewMockPhase("checkout", "git", "production-east-2"), nil
	})

	if err := checkout.Build(system); err != nil {
		return err
	}

	billing := pipelines.NewBuilder(config, glu.Name("billing"), NewMockResource).
		// oci phase
		NewPhase(func(pipelines.Builder[*MockResource]) (edges.Phase[*MockResource], error) {
			return NewMockPhase("billing", "oci", "oci"), nil
		}).
		// staging phase
		PromotesTo(func(pipelines.Builder[*MockResource]) (edges.UpdatablePhase[*MockResource], error) {
			return NewMockPhase("billing", "git", "staging"), nil
		}).
		// production phase
		PromotesTo(func(pipelines.Builder[*MockResource]) (edges.UpdatablePhase[*MockResource], error) {
			return NewMockPhase("billing", "git", "production"), nil
		})

	if err := billing.Build(system); err != nil {
		return err
	}

	return system.Run()
}

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("error running mock server", "error", err)
		os.Exit(1)
	}
}

// MockResource represents our mock resource state
type MockResource struct{}

func (m *MockResource) Digest() (string, error) {
	return "mock-digest", nil
}

func NewMockResource() *MockResource {
	return &MockResource{}
}

// MockPhase implements a simple phase for our mock resources
type MockPhase struct {
	pipeline, kind, name string
}

func NewMockPhase(pipeline, kind, name string) *MockPhase {
	return &MockPhase{pipeline: pipeline, kind: kind, name: name}
}

func (m *MockPhase) Descriptor() glu.Descriptor {
	return glu.Descriptor{
		Kind:     m.kind,
		Pipeline: m.pipeline,
		Metadata: core.Metadata{
			Name: m.name,
		},
	}
}

func (m *MockPhase) Get(ctx context.Context) (core.Resource, error) {
	return m.GetResource(ctx)
}

func (m *MockPhase) GetResource(context.Context) (*MockResource, error) {
	return &MockResource{}, nil
}

func (m *MockPhase) History(_ context.Context) ([]core.State, error) {
	return nil, nil
}

func (m *MockPhase) Update(_ context.Context, from, to *MockResource) (map[string]string, error) {
	return map[string]string{}, nil
}
