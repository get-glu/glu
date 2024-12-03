GitOps and Glu: Implementation
==============================

This guide follows on from the previous [GitOps and Glu: Overview](/guides/gitops-overview.md) guide.

In this guide, we will walkthrough the actual implementation (in Go) of the pipeline found in the [GitOps Example Repository](https://github.com/get-glu/gitops-example).

The pipeline code in particular is rooted in this directory: https://github.com/get-glu/gitops-example/tree/main/cmd/pipeline.

## The Goal

The goal of this codebase is to model and execute a promotion pipeline.
Whenever a new version of our application is pushed and tagged as `latest` in some source OCI repository we want to update our configuration Git repository, so that FluxCD can deploy it to our _staging_ environment.
Additionally, when a user decides the application in production is ready, we want them to be able to promote it to the _production_ environment.

Our example repository acts as the source of truth for FluxCD to apply to the target environments.
It contains the manifests that our pipeline is going to update for us.
The different `staging` and `production` directories in the `env` folder are where you will find these managed manifests:

```yaml
env
├── production
│   ├── deployment.yaml # our pipeline will update this based on staging
│   └── service.yaml
└── staging
    ├── deployment.yaml # our pipeline will update this based on a source OCI repository
    └── service.yaml
```

Our goal is to automate updating those `deployment.yaml` files in each environment specific directory.
In particular, we're going to update the pod spec's container image version:

```yaml
# ...
spec:
  containers:
    # ...
    image: ghcr.io/get-glu/gitops-example/app@sha256:d0ae5e10d27115a8bd245f989377c5eca2f74878b8a65bc4db21fd096551f802
# ...
```

Glu doesn't perform deployments directly (we're using FluxCD for that in this situation).
However, it instead keeps changes flowing in an orderly fashion from OCI through Git.

In Glu, we will:

1. Implement a _system_
2. Add a single release _pipeline_
3. Define three phases _oci_ (to track the source of new versions), _staging_ and _production_

At the end, we will add a trigger for the staging phase to attempt promotions based on a schedule.

## System

Every Glu codebase starts with a `glu.System`. A system is a container for your pipelines, an entrypoint for command line interactions, a server for running promotions, as well as a scheduler for triggering promotions based on conditions.

In the GitOps example, you will find a `main.go`. In this, you will find `main()` function, which calls a function `run(ctx) error`.
This function is where we first get introduced to our new glu system instance.

We start by creating a system and getting our pre-built configuration (this comes in handy configuring our phases later).

```go
func run(ctx context.Context) error {
    system := glu.NewSystem(ctx, glu.Name("gitops-example"), glu.WithUI(ui.FS()))
	config, err := system.Configuration()
	if err != nil {
		return err
	}

    // ...
}
```

The glu system needs some metadata (a container for a name and optional labels/annotations), as well as some optional extras.
`glu.Name` is a handy utility for creating an instance of `glu.Metadata` with the required field `name` set to the first argument passed.
We happen to want the UI, which is distributed as its own separate Go module:

```go
// go get github.com/get-glu/glu/ui@latest
import "github.com/get-glu/glu/ui"
```

> Keeping it as a separate module means that you don't have to include all the assets in your resulting pipeline to use Glu.
> The UI module bundles in a pre-built React/Typescript app as an instance of Go's `fs.FS`.

### Pipelines package

```go
import "github.com/get-glu/glu/pkg/pipelines"
```

Glu comes with a handy `pipelines` package for creating pipelines, which are strongly typed and easy to compose.
The package provides lots of useful convenience functions for quickly building integrations with e.g. Git and OCI.

Its purpose and goal is to provide a form of cached dependency injection for system and pipeline composition.

```go
pipelines.NewBuilder(config, glu.Name("gitops-example-app"), func() *AppResource {
    return &AppResource{
        Image: "ghcr.io/get-glu/gitops-example/app",
    }
})
```

> Notice that type called `*AppResource`?
> We won't get into this right now (we will learn about this in the [Resources](#resources) section).
> However, this type is intrinsic for defining _what_ flows through our pipeline and _how_ it is represented in our target **sources**.

The result of this function is a `*pipelines.PipelineBuilder`.
As we read on, the benefits of this type should hopefully become more apparent, over using the base system.

## Pipelines

In this example, we have a single pipeline we have chosen to call `gitops-example-app`.

> We tend to model our pipelines around the particular applications they deliver.

To add a new pipeline, we normally call `system.AddPipeline()`, which takes an instance of the `glu.Pipeline` interface:

```go
system.AddPipeline(glu.Pipeline)
```

However, our handy `pipelines.PipelineBuilder` has a final method `build.Build(system)` for registering the built pipeline at the end of the creating and adding all our phases:

```go
if err := pipelines.NewBuilder(config, glu.Name("gitops-example-app"), func() *AppResource {
    return &AppResource{
        Image: "ghcr.io/get-glu/gitops-example/app",
    }
// ...
}).Build(system); err != nil {
    return err
}
```

## Phases

Now that we have our pipeline named and our sources configured, we can begin defining our phases and their promotion dependencies.

Remember, we said that we plan to define three phases:

- OCI (the source phase)
- Staging (represented as configuration in our Git repository)
- Production (represented as configuration in our Git repository)

### OCI

```go
}).NewPhase(func(b pipelines.Builder[*AppResource]) (edges.Phase[*AppResource], error) {
    // build a phase which sources from the OCI repository
    return pipelines.OCIPhase(b, glu.Name("oci"), "app")
})
```

The initial phase is nice and simple. We're going to use the `NewPhase` method on the builder to make a new phase using our builder.
This method takes a function, which is provided with a builder type and expects you to return a typed phase.
Here we are using another utility function `pipelines.OCIPhase`.
This takes the builder we're provider, along with some metadata to identify it and a source name `"app"`.

#### Configuration

The `pipelines.OCIPhase` function builds a phase using a pre-configured OCI repository client using the system configuration and a name.

Notice we provided the source name `"app"` when creating the phase.
You can now provide OCI-specific configuration for this source repository by using the same name `"app"` in a `glu.yaml` (or this can alternatively be supplied via environment variables).

```yaml
sources:
  oci:
    app: # this is where our "app" argument comes in
      reference: "ghcr.io/get-glu/gitops-example/app:latest"
      credential: "github"

credentials:
  github:
    type: "basic"
    basic:
      username: "glu"
      password: "password"
```

### Git

The next two phases we're going to build will be backed by Git.
This is where we will configure access to configuration stored in respective paths in our repository.

#### Staging

```go
//...
.PromotesTo(func(b pipelines.Builder[*AppResource]) (edges.UpdatablePhase[*AppResource], error) {
    return pipelines.GitPhase(b, glu.Name("staging", glu.Label("url", "http://0.0.0.0:30081")), "gitopsexample")
})
```

Again, here we are building a new phase, passing it a function which takes a builder.
However, this time we did not use `NewPhase`, but instead used the handy `PromotesTo` method.
This has a similar signature to `NewPhase`, except instead we're chaining it directly after building the OCI phase.

This particular function both creates a phase and it creates a _promotion_ edge **to** the _staging_ phase **from** the OCI _phase_.
In other words, we make it so that you can promote from _oci_ to _staging_.
This is how we create the promotion paths from one phase to the next in Glu.

Here we're returning an `edges.UpdatePhase[*AppResource]` instead of just an `edges.Phase[*AppResource]`.
This interface includes an additional method to the phase abstraction, which means the phase is writable.
In order to support promoting to a phase, it needs to be writable so that we can update it.

Also, this time we use `pipelines.GitPhase` instead of `pipelines.OCIPhase`.
This gives us a phase which is backed by configuration in a Git Repository.
This function, similarly to OCI, takes a builder, some identifying metadata and a source name.

#### Production

```go
//...
.PromotesTo(func(b pipelines.Builder[*AppResource]) (edges.UpdatablePhase[*AppResource], error) {
    return pipelines.GitPhase(b, glu.Name("production", glu.Label("url", "http://0.0.0.0:30082")), "gitopsexample")
})
```

Finally, we describe our _production_ phase. As with staging, we pass the builder, metadata, a source name and we chain it onto the staging phase instead.
This means we can promote from _staging_ to _production_.

Now we have described our entire end-to-end _pipeline_.

#### Configuration

As with OCI, Git has a similar convenience function in the `pipelines` package for getting a pre-configured Git client.
Here, we ask for a git phase with the name name `"gitopsexample"`.

Again, as with the OCI phase, we can now configure this named repository in our `glu.yaml` or via environment variables.

```yaml
sources:
  git:
    gitopsexample: # this is where our "gitopsexample" argument comes in
      remote:
        name: "origin"
        url: "https://github.com/get-glu/gitops-example.git"
        credential: "github"

credentials:
  github:
    type: "basic"
    basic:
      username: "glu"
      password: "password"
```

## Resources

Remember that `*AppResource` type that has been floating around?

This particular piece of the puzzle has two goals in the system:

1. What are we promoting?

2. How is it represented in _each_ source?

### The What

```go
// AppResource is a custom envelope for carrying our specific repository configuration
// from one source to the next in our pipeline.
type AppResource struct {
	Image       string
	ImageDigest string
}

// Digest is a core required function for implementing glu.Resource.
// It should return a unique digest for the state of the resource.
// In this instance we happen to be reading a unique digest from the source
// and so we can lean into that.
// This will be used for comparisons in the phase to decide whether or not
// a change has occurred when deciding if to update the target source.
func (c *AppResource) Digest() (string, error) {
	return c.ImageDigest, nil
}
```

In our particular GitOps repository, the _what_ is OCI image digests.
We're interested in updating an application (bundled into an OCI repository) version across different target environments.

By defining a type that implements the `core.Resource` interface, we can use it in our pipeline.

```go
// Resource is an instance of a resource in a phase.
// Primarily, it exposes a Digest method used to produce
// a hash digest of the resource instances' current state.
type Resource interface {
	Digest() (string, error)
}
```

A resource (currently) only needs a single method `Digest()`.
It is up to the implementer to return a string, which is a content digest of the resource itself.
In our example, we return the actual image digest field, as this is the unique digest we're using to make promotion decision with.

> This is used for comparison when making promotion decisions.
> If two instances of your resources (i.e. the version in OCI, compared with the version in staging) differ, then a promotion will take place.

### The How

This part is important, and the functions you need to implement depend on the sources you're using.
Whenever you attempt to integrate a source into a phase for a given resource type, the source will add further compile constraints.

#### oci.Phase[Resource]

```go
type Resource interface {
	core.Resource
	ReadFromOCIDescriptor(v1.Descriptor) error
}
```

The OCI phase (currently a read-only phase) requires a method `ReadFromOCIDescriptor(...)`.
By implementing this method, you can combine your resource type and this source type into a phase for your pipeline.
The method should extract any necessary details onto your type structure.
These details should be the ones that change and are copied between phases.

```go
// ReadFromOCIDescriptor is an OCI-specific resource requirement.
// Its purpose is to read the resources state from a target OCI metadata descriptor.
// Here we're reading out the images digest from the metadata.
func (c *AppResource) ReadFromOCIDescriptor(d v1.Descriptor) error {
	c.ImageDigest = d.Digest.String()
	return nil
}
```

In our example, we're interested in moving the OCI image digest from one phase to the next.
So, we extract that from the image descriptor provided to this function and set it on the field we defined on our type.

#### git.Phase[Resource]

```go
type Resource interface {
	core.Resource
	ReadFrom(context.Context, core.Metadata, fs.Filesystem) error
	WriteTo(context.Context, core.Metadata, fs.Filesystem) error
}
```

The Git source requires a resource to be readable from and writeable to a target filesystem.
This source is particularly special, as it takes care of details such as checking out branches, staging changes, creating commits, opening pull requests, and so on.
Instead, all it requires the implementer to do is explain how to read and write the definition to a target repository root tree.
The source then takes care of the rest of the contribution lifecycle.

> There are also further ways to configure the resulting commit message, PR title, and body via other methods on your type.

**GitOps Example: Reading from filesystem**

```go
func (r *AppResource) ReadFrom(_ context.Context, phase core.Descriptor, fs fs.Filesystem) error {
	deployment, err := readDeployment(fs, fmt.Sprintf("env/%s/deployment.yaml", phase.Metadata.Name))
	if err != nil {
		return err
	}

	if containers := deployment.Spec.Template.Spec.Containers; len(containers) > 0 {
		ref, err := registry.ParseReference(containers[0].Image)
		if err != nil {
			return err
		}

		digest, err := ref.Digest()
		if err != nil {
			return err
		}

		r.ImageDigest = digest.String()
	}

	return nil
}
```

Here we see that we read a file at a particular path: `fmt.Sprintf("env/%s/deployment.yaml", phase.Descriptor.Name)`

The descriptor supplied by Glu here happens to be the phases identifiers and metadata.
This is how we can read different paths, dependent on the phase being read or written to.

This particular implementation reads the file as a Kubernetes deployment encoded as YAML.
It then extracts the container's image reference directly from the pod spec.

The resulting image digest is again set on the receiving resource type `c.ImageDigest = digest.String()`.

**GitOps Example: Writing to filesystem**

```go
func (r *AppResource) WriteTo(ctx context.Context, phase glu.Descriptor, fs fs.Filesystem) error {
	path := fmt.Sprintf("env/%s/deployment.yaml", phase.Descriptor.Name)
	deployment, err := readDeployment(fs, path)
	if err != nil {
		return err
	}

    // locate the target container
	if containers := deployment.Spec.Template.Spec.Containers; len(containers) > 0 {
        // update containers image <repository>@<digest>
		containers[0].Image = fmt.Sprintf("%s@%s", c.Image, c.ImageDigest)

        // update an informative environment variable with the digest too
		for i := range containers[0].Env {
			if containers[0].Env[i].Name == "APP_IMAGE_DIGEST" {
				containers[0].Env[i].Value = r.ImageDigest
			}
		}
	}

    // re-open the file for writing
	fi, err := fs.OpenFile(
		path,
		os.O_WRONLY|os.O_TRUNC,
		0644,
	)
	if err != nil {
		return err
	}

	defer fi.Close()
    
    // re-encode the deployment and copy it into the file
	data, err := yaml.Marshal(deployment)
	if err != nil {
		return err
	}

	_, err = io.Copy(fi, bytes.NewReader(data))
	return err
}
```

When it comes to writing, again we look to the file in the path dictated by metadata from the phase.

Here, we're taking the image digest from our receiving `r *AppResource` and setting it on the target container in our deployment pod spec.
Finally, we're rewriting our target file with the newly updated contents of our deployment.

## The Triggers

Triggers are used to automate promotions based on some event or schedule.
Currently, scheduled triggers are the only prebuilt triggers available (we intend to add more, e.g. on oci or git push).

```go
system.AddTrigger(
	schedule.New(
		schedule.WithInterval(10*time.Second),
		schedule.MatchesLabel("env", "staging"),
		// alternatively, the phase instance can be target directly with:
		// glu.ScheduleMatchesPhase(stagingPhase),
	),
)
```

Here, we configure the system to attempt a promotion on any phase with a particular label pair (`"env" == "staging"`) every `10s`.
Remember, a phase will only perform a real promotion if the resource derived from the two sources differs (based on comparing the result of `Digest()`).

## Now Run

Finally, we can now run our pipeline.

```go
system.Run()
```

This is usually the last function call in a Glu system binary.
It takes care of setting up signal traps and propagating terminations through context cancellation to the various components.

Additionally, if you have configured any triggers and are invoking your glu pipeline as a server binary, then these will be enabled for the duration of the process.

## Recap

OK, that was a lot of concepts to learn (hopefully, not too much code to write (~<200 LOC)).
I appreciate you taking the time to walk through this journey!

We have:

- Created a glu system to orchestrate our promotion pipelines with
- Declared three phases (based over OCI and Git repositories) to promote change between
- Defined a resource type for carrying our promotion material through the pipeline

The byproduct of doing all this is that we get an instant API for introspecting and promoting changes through pipelines.
Promotions can be triggered manually through the UI and API, or configured to run on a schedule.
All changes flow in and out of our disparate sources, via whichever encoding formats/configuration languages and filesystem layouts we choose.
