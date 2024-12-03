# Concepts

Glu has an opinionated set of models and abstractions, which when combined, allow you to build consistent command-line and server processes for orchestrating the progression of applications and configuration across target environments.

```go
if err := pipelines.NewBuilder(config, glu.Name("checkout"), NewCheckoutResource).
        // build an OCI phase from the OCI source named "checkout"
		NewPhase(func(b pipelines.Builder[*CheckoutResource]) (edges.Phase[*CheckoutResource], error) {
			return pipelines.OCIPhase(b, glu.Name("oci"), "checkout")
		}).
        // build a phase for the staging environment from the git repo source named "checkout"
        // and configure it to promote from the OCI phase
		PromotesTo(func(b pipelines.Builder[*CheckoutResource]) (edges.UpdatablePhase[*CheckoutResource], error) {
			return pipelines.GitPhase(b, glu.Name("staging", glu.Label("env", "staging")), "checkout",
				git.ProposeChanges[*CheckoutResource](git.ProposalOption{
					Labels: []string{"automerge"},
				}))
		}, schedule.New(
            // configure the promotion to run every 30 seconds
			schedule.WithInterval(30*time.Second),
		)).
        // build a phase for the production environment from the git repo source named "checkout"
        // and configure it to promote from the staging git phase
		PromotesTo(func(b pipelines.Builder[*CheckoutResource]) (edges.UpdatablePhase[*CheckoutResource], error) {
			return pipelines.GitPhase(b, glu.Name("production", glu.Label("env", "production")), "checkout")
		}).
		Build(system); err != nil {
		return err
	}
```

The Glu framework comprises of a set of abstractions for declaring the resources (your applications and configuration), update strategies (we call them phases) and rules for progression (how and when to promote) within a pipeline.

### System

At the top of the tree we have the `glu.System`.
This is our entrypoint and wrapper for building a Glu configured application.

```go
func main() {
    system := glu.NewSystem(context.Background(), glu.Name("mysystem"))
    // ...
    if err := system.Run(); err != nil {
        panic(err)
    }
}
```

The `glu.System` instance has methods for building new pipelines and configuring triggers.

By delegating control to a `glu.System` instance we get a few out-of-the-box conveniences:

1. Pre-built configuration format and parsing for accessing and authenticating the built-in sources (Git, OCI and so on).
1. A CLI interface for inspecting and manually promoting resources on the command-line.
1. An API interface for inspecting and manually promoting resources over a network.
1. Lifecycle control (trigger scheduling, signal handling and graceful shutdown).
1. Add the optional UI component to visualize your pipelines in a browser.

### Resources

Resources are the primary definition of _what_ is being represented in your pipeline and _how_ they are represented in target sources.
In Go, resources are represented as an interface for the author (that's you) to fill in the blanks:

```go
// Resource is an instance of a resource in a phase.
// Primarily, it exposes a Digest method used to produce
// a hash digest of the instances current state.
type Resource interface {
	Digest() (string, error)
}
```

The core abstraction (currently) only requires the type to implement a single `Digest()` method.
This should be used to produce a content digest of the resources state in a given moment.
It is used in the system to perform equality checks when deciding whether or not to promote.

The other sources in the Glu codebase will require additional functions to be implemented in order to integrate with them.
For example, the Git source requires you to define how your resource is encoded and decoded to and from a filesystem.

```go
type SomeResource struct {
    ImageName   string `json:"image_name"`
    ImageDigest string `json:"image_digest"`
}

func (s *SomeResource) Digest() (string, error) { return s.ImageDigest, nil }
```

### Pipelines

Pipelines carry your specific resources type across phase destinations.
They coordinate the edges in the workflow that is your promotion pipeline.

They are also responsible for constructing new instances of your resource types.
These are used when fetching and updating phases and sources during promotion.

```go
pipeline := glu.NewPipeline(glu.Name("mypipeline"), func() *SomeResource {
    // this is an opportunity to set any initial default values
    // for fields before the rest of the fields are populated
    // from a target phase source
    return &SomeResource{ImageName: "someimagename"}
})
```

### Phases

Phases are the core engine for both viewing and updating resources in a target external system.
They interface with your resource types and storage mediums to perform relevant transactions.

Currently, Glu has implementations for the following sources:

- [OCI](../pkg/phases/oci)
- [Git](../pkg/phases/git)

We look to add more in the not-so-distant future. However, these can also be implemented by hand via the following interfaces:

```go
// Phase is the core interface for resource sourcing and management.
// These types can be registered on pipelines and can depend upon on another for promotion.
type Phase interface {
	Descriptor() Descriptor
	Get(context.Context) (Resource, error)
	History(context.Context) ([]State, error)
}
```

### Edges

Edges have the job of interfacing with source and destination phases.
They connect one phase to another as a directed dependency in the pipeline graph.
Edges also perform some action between the two phases expressed via a method `Perform()`:

```go
// Edge represents an edge between two phases.
// Edges have have their own kind which identifies their Perform behaviour.
type Edge interface {
	Kind() string
	From() Descriptor
	To() Descriptor
	Perform(context.Context) (Result, error)
	CanPerform(context.Context) (bool, error)
}
```

#### Promotion

The core `promotion` kind edge promotes one phase to the next on a call to `Perform(ctx)`.
The process is as follows:

1. Get the current resource state from the source phase based on phase metadata.
2. Get the current resource state from the destination promotion target phase.
3. If the resource from (2) is equal to that of (3) (based on comparing their digest), then return (no-op).
4. Update the state of the phase destination with the state of the upstream resource (3) (this is a promotion).

### Triggers

Edges can be decorated so that their `Perform` method is invoked automatically under certain conditions.
For example, they can be performed on a schedule:

```go
// schedule a promotion attempt every 10 seconds
pipeline.AddEdge(triggers.Edge(
    promotionEdge,
	schedule.New(
		schedule.WithInterval(10*time.Second),
	)
))
```
