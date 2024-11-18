# Guide

It's perhaps easiest to understand Glu by example.

## Example Pipeline

In this example, we'll create a pipeline consisting of four phases:

1. **OCI**: Fetch the OCI artifact from a remote registry.
2. **Staging**: Promote the OCI artifact to the staging environment.
3. **Production-East-1**: Promote the OCI artifact to the production environment in the east-1 region.
4. **Production-West-1**: Promote the OCI artifact to the production environment in the west-1 region.

Before we begin, we'll need to define a `Resource` type which represents the resource we're promoting.

Here we define a `CheckoutResource` type which will be used to carry our specific repository configuration
from one source to the next in our pipeline.

```go
type CheckoutResource struct {
	ImageDigest string `json:"digest"`
}

func NewCheckoutResource() *CheckoutResource {
	return &CheckoutResource{}
}

// Digest is a core required function for implementing glu.Resource
// It should return a unique digest for the state of the resource.
// This will be used for comparisons in the phase to decided whether or not
// a change has occurred when deciding if to update the target source.
func (c *CheckoutResource) Digest() (string, error) {
	return c.ImageDigest, nil
}
```

Here is how you can create this pipeline using Glu:

```go
system := glu.NewSystem(ctx, glu.Name("mycorp", glu.Label("team", "ecommerce")))
	system.AddPipeline(func(ctx context.Context, config *glu.Config) (glu.Pipeline, error) {
		pipeline := glu.NewPipeline(glu.Name("checkout"), NewCheckoutResource)

		// fetch the configured OCI repositority source named "checkout"
		ociRepo, err := config.OCIRepository("checkout")
		if err != nil {
			return nil, err
		}
	
		// OCI phase
		ociPhase, err := phases.New(
			glu.Name("oci"),
			pipeline,
			oci.New[*CheckoutResource](ociRepo),
		)
		if err != nil {
			return nil, err
		}

		// fetch the configured Git repository source named "checkout"
		gitRepo, _, err := config.GitRepository(ctx, "checkout")
		if err != nil {
			return nil, err
		}

		// create a new Git source for the pipeline which will be used to
		// read and write to the remote repository via the phase.
		gitSource := git.NewSource[*CheckoutResource](gitRepo, nil)

		// Staging phase
		stagingPhase, err := phases.New(
			glu.Name("staging",
				glu.Label("environment", "staging"),
				glu.Label("region", "us-east-1"),
			),
			pipeline,
			gitSource,
			core.PromotesFrom(ociPhase),
		)
		if err != nil {
			return nil, err
		}

		// Production phases
		phases.New(
			glu.Name("production-east-1",
				glu.Label("environment", "production"),
				glu.Label("region", "us-east-1"),
			),
			pipeline,
			gitSource,
			core.PromotesFrom(stagingPhase),
		)

		phases.New(
			glu.Name("production-west-1",
				glu.Label("environment", "production"),
				glu.Label("region", "us-west-1"),
			),
			pipeline,
			gitSource,
			core.PromotesFrom(stagingPhase),
		)

		return pipeline, nil
	})
)
```

Let's break down what we've done here.

1. We've created a new system with a name and label.
2. We've added a pipeline with a `CheckoutResource` resource type. This allows the pipeline to create new instances of `CheckoutResource` for each run.
3. We've added four phases to the pipeline: `oci`, `staging`, `production-east-1`, and `production-west-1`.
4. We instantiated a repository source for the `oci` phase. This source is responsible for fetching the OCI artifact from a remote registry.
5. We've added a source for the `staging`, `production-east-1`, and `production-west-1` phases. These sources will be used to check if the OCI artifact digest has changed upstream. If it has, the pipeline will promote the artifact to the next phase.

## Inspecting The Pipeline

Glu provides two main ways to inspect the pipeline and its state.

### CLI

The glu package automatically generates a CLI to manually inspect your system and pipelines.

```sh
go run . inspect checkout
```

```console
NAME                DEPENDS_ON
oci                 
staging             oci
production-east-1   staging
production-west-1   staging
```

You can also inspect the state of a specific phase.

```sh
go run . inspect checkout staging
```

```console
NAME                DIGEST
staging             sha256:5338d5b9949c5e94924b28dbca563a968ae7694209686e5e71ed8fa10c11fce0
```

This will show the current state of the pipeline resource in the `staging` phase for the `checkout` pipeline.

### UI

