package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/get-glu/glu"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/get-glu/glu/pkg/phases"
	"github.com/get-glu/glu/pkg/src/git"
	"github.com/get-glu/glu/pkg/src/oci"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gopkg.in/yaml.v3"
)

func run(ctx context.Context) error {
	return glu.NewSystem(ctx).AddPipeline(func(ctx context.Context, config *glu.Config) (glu.Pipeline, error) {
		// fetch the configured OCI repositority source named "checkout"
		ociRepo, err := config.OCIRepository("checkout")
		if err != nil {
			return nil, err
		}

		ociSource := oci.New[*CheckoutResource](ociRepo)

		// fetch the configured Git repository source named "checkout"
		gitRepo, gitProposer, err := config.GitRepository(ctx, "checkout")
		if err != nil {
			return nil, err
		}

		gitSource := git.NewSource(gitRepo, gitProposer, git.ProposeChanges[*CheckoutResource](git.ProposalOption{
			Labels: []string{"automerge"},
		}))

		// create initial (empty) pipeline
		pipeline := glu.NewPipeline(glu.Name("checkout"), NewCheckoutResource)

		// build a phase which sources from the OCI repository
		ociPhase, err := phases.New(glu.Name("oci"), pipeline, ociSource)
		if err != nil {
			return nil, err
		}

		// build a phase for the staging environment which source from the git repository
		// configure it to promote from the OCI phase
		staging, err := phases.New(glu.Name("staging", glu.Label("env", "staging")),
			pipeline, gitSource, core.PromotesFrom(ociPhase))
		if err != nil {
			return nil, err
		}

		// build a phase for the production environment which source from the git repository
		// configure it to promote from the staging git phase
		_, err = phases.New(glu.Name("production", glu.Label("env", "production")),
			pipeline, gitSource, core.PromotesFrom(staging))
		if err != nil {
			return nil, err
		}

		// return configured pipeline to the system
		return pipeline, nil
	}).SchedulePromotion(
		glu.ScheduleInterval(10*time.Second),
		glu.ScheduleMatchesLabel("env", "staging"),
		// alternatively, the phase instance can be target directly with:
		// glu.ScheduleMatchesPhase(gitStaging),
	).Run()
}

// CheckoutResource is a custom envelope for carrying our specific repository configuration
// from one source to the next in our pipeline.
type CheckoutResource struct {
	ImageDigest string `json:"digest"`
}

// NewCheckoutResource constructs a new instance of the CheckoutResource.
// This function is required for creating a new pipeline.
func NewCheckoutResource() *CheckoutResource {
	return &CheckoutResource{}
}

// Digest is a core required function for implementing glu.Resource
// It should return a unique digest for the state of the resource.
// In this instance we happen to be reading a unique digest from the source
// and so we can lean into that.
// This will be used for comparisons in the phase to decided whether or not
// a change has occurred when deciding if to update the target source.
func (c *CheckoutResource) Digest() (string, error) {
	return c.ImageDigest, nil
}

// CommitMessage is an optional git specific method for overriding generated commit messages.
// The function is provided with the source phases metadata and the previous value of resource.
func (c *CheckoutResource) CommitMessage(meta glu.Metadata, _ *CheckoutResource) (string, error) {
	return fmt.Sprintf("feat: update app %q in %q", meta.Name, meta.Labels["env"]), nil
}

// ProposalTitle is an optional git specific method for overriding generated proposal message (PR/MR) title message.
// The function is provided with the source phases metadata and the previous value of resource.
func (c *CheckoutResource) ProposalTitle(meta glu.Metadata, r *CheckoutResource) (string, error) {
	return c.CommitMessage(meta, r)
}

// ProposalBody is an optional git specific method for overriding generated proposal body (PR/MR) body message.
// The function is provided with the source phases metadata and the previous value of resource.
func (c *CheckoutResource) ProposalBody(meta glu.Metadata, r *CheckoutResource) (string, error) {
	return fmt.Sprintf(`| app | from | to |
| -------- | ---- | -- |
| checkout | %s | %s |
`, r.ImageDigest, c.ImageDigest), nil
}

// ReadFromOCIDescriptor is an OCI specific resource requirement.
// Its purpose is to read the resources state from a target OCI metadata descriptor.
// Here we're reading out the images digest from the metadata.
func (c *CheckoutResource) ReadFromOCIDescriptor(d v1.Descriptor) error {
	c.ImageDigest = d.Digest.String()
	return nil
}

// ReadFrom is a Git specific resource requirement.
// It specifies how to read the resource from a target Filesystem.
// The type should navigate and source the relevant state from the fileystem provided.
// The function is also provided with metadata for the calling phase.
// This allows the defining type to adjust behaviour based on the context of the phase.
// Here we are reading a yaml file from a directory signified by a label ("env") on the phase metadata.
func (c *CheckoutResource) ReadFrom(_ context.Context, meta core.Metadata, fs fs.Filesystem) error {
	fi, err := fs.OpenFile(
		fmt.Sprintf("/env/%s/apps/checkout/deployment.yaml", meta.Labels["env"]),
		os.O_RDONLY,
		0644,
	)
	if err != nil {
		return err
	}

	defer fi.Close()

	var manifest struct {
		Digest string
	}
	if err := yaml.NewDecoder(fi).Decode(&manifest); err != nil {
		return err
	}

	c.ImageDigest = manifest.Digest

	return nil
}

// WriteTo is a Git specific resource requirement.
// It specifies how to write the resource to a target Filesystem.
// The type should navigate and encode the state of the resource to the target Filesystem.
// The function is also provided with metadata for the calling phase.
// This allows the defining type to adjust behaviour based on the context of the phase.
// Here we are writing a yaml file to a directory signified by a label ("env") on the phase metadata.
func (c *CheckoutResource) WriteTo(ctx context.Context, meta glu.Metadata, fs fs.Filesystem) error {
	fi, err := fs.OpenFile(
		fmt.Sprintf("/env/%s/apps/checkout/deployment.yaml", meta.Labels["env"]),
		os.O_RDONLY,
		0644,
	)
	if err != nil {
		return err
	}

	defer fi.Close()

	return yaml.NewEncoder(fi).Encode(c)
}
