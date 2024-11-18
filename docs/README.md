# Welcome <!-- {docsify-ignore} -->

> Glu is progressive delivery as code

Glu is a framework written and consumed in the Go programming language.
It is intended to glue together configuration repositories and artifact sources using `code`.

## Goals

The goals outlined here are defined to set a target for the project to work towards.
Glu has not yet reached all of these goals, but hopefully it gives an idea of our intended direction.
They're also not immutable and we would love for your help in shaping them.

- Allow engineers to define application delivery across multiple environments as `code`.
- Create a central pane of glass for inspecting and interacting with promotion pipelines.
- Support defining complex rules to guard automated promotion using strategies such as smoke testing and performance evaluation.
- Simple and quick integration with common existing CI/CD and source control management platforms.
- Expose standardized telemetry for the entire end-to-end delivery of changes in your system.

## Overview

Glu has an opinionated set of models and abstractions, which when combined, allow you to build consistent command-line and server processes for orchestrating the progression of applications and configuration across target environments.

```go
// build a phase which sources from our OCI repository
ociPhase, err := phases.New(glu.Name("oci"), pipeline, ociSource)
if err != nil {
    return nil, err
}

// build a phase for the staging environment which sources from our git repository
// and configures it to promote from the OCI phase
staging, err := phases.New(glu.Name("staging", glu.Label("env", "staging")),
	pipeline, gitSource, core.PromotesFrom(ociPhase))
if err != nil {
	return nil, err
}

// build a phase for the production environment which also sources from our git repository
// and configures it to promote from the staging git phase
_, err = phases.New(glu.Name("production", glu.Label("env", "production")),
	pipeline, gitSource, core.PromotesFrom(staging))
if err != nil {
	return nil, err
}
```

The Glu framework comprises of a set of abstractions for declaring the resources (your applications and configuration), update strategies (we call them phases) and rules for progression (how and when to promote) within a pipeline.

### System

At the top of the tree we have the `glu.System`.
This is our entrypoint and wrapper for building a Glu configured application.

```go
func main() {
    glu.NewSystem(context.Background()).Run()
}
```

The `glu.System` instance has methods for building new pipelines and configuring triggers.

By delegating control to a `glu.System` instance we get a few out-of-the-box conveniences:

1. Pre-built configuration format and parsing for accessing and authenticating the built-in sources (Git, OCI and so on).
1. A CLI interface for inspecting and manually promoting resources on the command-line.
1. An API interface for inspecting and manually promoting resources over a network.
1. Lifecycle control (signal handling and graceful shutdown).
1. Add the optional UI component to visualize your pipelines in a browser.

### Resources

Resources are the primary definition of _what_ is being represented in your pipeline and _how_ they are represented in target sources.
In Go, resources are represented as an interface for the author (that's you) to fill in the blanks:

```go
// Resource is an instance of a resource in a phase.
// Primarilly, it exposes a Digest method used to produce
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

Phases have the job of interfacing with both sources and other upstream phases to manage the promotion lifecycle.
They have metadata to uniquely identify themselves within the context of a pipeline.
They're also bound to a particular source implementation, and are optional dependent on an upstream source of the _same resource type_.

When a phase attempts a promotion (`phase.Promote(ctx)`) the following occurs:

1. If there is no upstream phase to promote from, then return (no op).
2. Get the current resource state from the target source based on phase metadata.
3. Get the current resource state from the upstream promotion target phase.
4. If the resource from (2) is equal to that of (3) (based on comparing their digest), the return (no op).
5. Update the state of the source with the state of the upstream resource (3) (this is a promotion).

### Sources

Sources are the core engine for phases to both view and update resources in a target external system.
While phases handle the lifecycle of promotion, sources interface with your resource types and sources of truth to perform relevant transactions.

Currently, Glu has implementations for the following sources:

- OCI
- Git

We look to add more in the not-so-distant future. However, these can also be implemented by hand via the following interfaces:

```go
// Phase is the core interface for resource sourcing and management.
// These types can be registered on pipelines and can depend upon on another for promotion.
type Phase interface {
	Metadata() Metadata
	Get(context.Context) (any, error)
	Promote(context.Context) error
}

// ResourcePhase is a Phase bound to a Resource type R.
type ResourcePhase[R Resource] interface {
	Phase
	GetResource(context.Context) (R, error)
}
```

### Triggers

Schedule promotions to run automatically on an interval for phases matching a specific set of labels:

```go
// schedule promotion attempts to staging every 10 seconds
system.SchedulePromotion(
    glu.ScheduleInterval(10*time.Second),
    glu.ScheduleMatchesLabel("env", "staging"),
)
```
