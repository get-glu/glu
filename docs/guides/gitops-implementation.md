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

Glu doesn't perform deployments directly (we're using FluxCD for that in this situation).
However, it instead keeps changes flowing in an orderly fashion from OCI through Git.

In Glu, we will implement a _system_, with a single release _pipeline_, consisting of three phases _oci_ (to mode the source of new versions), _staging_ and _production_.

At the end, we will add a trigger for the staging phase to attempt promotions based on a schedule.

## The System

Every Glu codebase starts with a `glu.System`. A system is a container for your pipelines, and entrypoint for command line interactions and starting the built-in server, as well as a scheduler for running promotions based on triggers.

In the GitOps example, you will find a `main.go`. In this, you will find `main()` function, which calls a function `run(ctx) error`.

This `run()` function is where we first get introduced to our new glu system instance.

```go
func run(ctx context.Context) error {
	return glu.NewSystem(ctx, glu.Name("gitops-example"), glu.WithUI(ui.FS())).
    // ...
```

A glu system needs some metadata (a container for a name and optional labels/annotations), as well as some optional extras.
`glu.Name` is a handy utility for creating an instance of `glu.Metadata` with the required field `name` set to the first argument passed.
We happen to want the UI, which is distributed as its own separate Go module:

```go
// go get github.com/get-glu/glu/ui@latest
import "github.com/get-glu/glu/ui"
```

> Keeping it as a separate module means that you don't have to include all the assets in your resulting pipeline to use Glu.
> The UI module bundles in a pre-built React/Typescript app as an instance of Go's `fs.FS`.

The `System` type exposes a few functions for adding new pipelines (`AddPipeline`), declaring triggers (`AddTrigger`), and running the entire system (`Run`).

## The Pipeline

In this example, we have a single pipeline we have chosen to call `gitops-example-app`.

> We tend to model our pipelines around the particular applications they deliver.

To add a new pipeline, we call `AddPipeline()`, which takes a function that returns a Pipeline for some provided configuration:

```go
system.AddPipeline(glu.BuilderFunc(func(builder *glu.PipelineBuilder[*AppResource]) (glu.Pipeline, error) {
  // ...
}))
```

It is up to the implementer to build a pipeline using the provided configuration.
The returned pipeline from this function will be registered on the system.
In this function, we're expected to define our pipeline and all its child phases and their promotion dependencies on one another.

Here, we're using a utility function `glu.BuilderFunc`.
This has some nice out of the box typed wrappers for quickly getting configured Git and OCI source clients.
Notice this type called `*AppResource`. We won't get into this right now (we will learn about this in [The Resource](#the-resource) section), however, this type is intrinsic for defining _what_ flows through our pipeline and _how_ it is represented in our target **sources**.

### Definition

```go
pipeline := glu.NewPipeline(glu.Name("gitops-example-app"), func() *AppResource {
    return &AppResource{
        Image: "ghcr.io/get-glu/gitops-example/app",
    }
})
```

As with our system, pipelines require some metadata.

Before we can define our phases, each phase will likely need to source its state from somewhere (e.g. OCI or Git).
We can use the config argument passed to the builder function to make this process easier.
Notice this function also takes a function which returns our `*AppResource` from before.
The pipeline will use this function to instantiate new, default versions of our resource when fetching state from sources.

### Config: OCI

```go
// fetch the configured OCI source named "app"
ociSource, err := glu.OCISource(builder, "app")
if err != nil {
    return nil, err
}
```

The `OCISource` method fetches a pre-configured OCI source client.
Notice we provide the name `"app"` when creating the repository.
You can now provide OCI-specific configuration for this source repository by using the same name `"app"` in a `glu.yaml` (or this can alternatively be supplied via environment variables).
This is key to how Glu brings conventions around configuration.

```yaml
sources:
  oci:
    app: # this is where our "app" name argument comes in
      reference: "ghcr.io/get-glu/gitops-example/app:latest"
      credential: "github"

credentials:
  github:
    type: "basic"
    basic:
      username: "glu"
      password: "password"
```

### Config: Git

```go
// fetch the configured Git repository source named "gitopsexample"
gitSource, err := config.GitSource(builder, "gitopsexample")
if err != nil {
    return nil, err
}
```

As with OCI, Git has a similar convenience function for getting a pre-configured Git source.
Here, we ask for configuration for a git source configured with the name `"gitopsexample"`.
Again, as with the OCI source, we can now configure this named repository in our `glu.yaml` or via environment variables.

```yaml
sources:
  git:
    gitopsexample: # this is where our config.GitRepository("gitopsexample") argument comes in
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

## The Phases

Now that we have our pipeline named and our sources configured, we can begin defining our phases and their promotion dependencies.

Remember, we said that we plan to define three phases:

- OCI (the source phase)
- Staging (represented as configuration in our Git repository)
- Production (represented as configuration in our Git repository)

### OCI

```go
// build a phase which sources from the OCI repository
ociPhase, err := phases.New(glu.Name("oci"), pipeline, ociSource)
if err != nil {
    return nil, err
}
```

The initial phase is nice and simple. We're going to use the `phases` package to make a new phase using our pipeline and our source.
As with our pipeline and system, we need to give it some metadata (name and optional labels/annotations).

### Staging (Git)

```go
// build a phase for the staging environment which sources from the git repository
// configure it to promote from the OCI phase
stagingPhase, err := phases.New(glu.Name("staging", glu.Label("url", "http://0.0.0.0:30081")),
    pipeline, gitSource, core.PromotesFrom(ociPhase))
if err != nil {
    return nil, err
}
```

Again, here we give the phase some metadata (this time with a helpful label) and pass additional dependencies (pipeline and git source).
However, we also now pass a new option `core.PromotesFrom(ociPhase)`.
This particular option creates a _promotion_ dependency to the _staging_ phase from the OCI _phase_.
In other words, we make it so that you can promote from _oci_ to _staging_.

This is how we create the promotion paths from one phase to the next in Glu.

### Production (Git)

```go
// build a phase for the production environment which sources from the git repository
// configure it to promote from the staging git phase
_, err = phases.New(glu.Name("production", glu.Label("url", "http://0.0.0.0:30082")),
    pipeline, gitSource, core.PromotesFrom(stagingPhase))
if err != nil {
    return nil, err
}
```

Finally, we describe our _production_ phase. As with staging, we pass metadata, pipeline, the git source and this time we add a promotion relationship to the `stagingPhase`. This means we can promote from _staging_ to _production_.

Now we have described our entire end-to-end _phase_.
However, it is crucial to now understand more about the `*AppResource`.

## The Resource

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

#### oci.Source[Resource]

```go
type Resource interface {
	core.Resource
	ReadFromOCIDescriptor(v1.Descriptor) error
}
```

The OCI source (currently a read-only phase source) requires a single method `ReadFromOCIDescriptor(...)`.
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

#### git.Source[Resource]

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

> There are further ways to configure the resulting commit message, PR title, and body via other methods on your type.

**GitOps Example: Reading from filesystem**

```go
func (r *AppResource) ReadFrom(_ context.Context, meta core.Metadata, fs fs.Filesystem) error {
	deployment, err := readDeployment(fs, fmt.Sprintf("env/%s/deployment.yaml", meta.Name))
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

Here we see that we read a file at a particular path: `fmt.Sprintf("env/%s/deployment.yaml", meta.Name)`

The metadata supplied by Glu here happens to be the phases metadata.
This is how we can read different paths, dependent on the phase being read or written to.

This particular implementation reads the file as a Kubernetes deployment encoded as YAML.
It then extracts the container's image reference directly from the pod spec.

The resulting image digest is again set on the receiving resource type `c.ImageDigest = digest.String()`.

**GitOps Example: Writing to filesystem**

```go
func (r *AppResource) WriteTo(ctx context.Context, meta glu.Metadata, fs fs.Filesystem) error {
	path := fmt.Sprintf("env/%s/deployment.yaml", meta.Name)
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

OK, that was a lot. I appreciate you taking the time to walk through this journey!

We have:

- Created a glu system to orchestrate our promotion pipelines with
- Configured two sources (OCI and Git) for reading and writing to
- Declared three phases to promote change between
- Defined a resource type for carrying our promotion material through the pipeline

The byproduct of doing all this is that we get an instant API for introspecting and manually promoting changes through pipelines.
All changes flow in and out of our disparate sources, via whichever encoding formats/configuration languages and filesystem layouts we choose.
Additionally, we can add a dashboard interface for humans to read and interact with, in order to trigger manual promotions.
