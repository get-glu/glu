package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/get-glu/glu"
	"github.com/get-glu/glu/pkg/edges"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/get-glu/glu/pkg/phases/git"
	"github.com/get-glu/glu/pkg/phases/oci"
	"github.com/get-glu/glu/pkg/pipelines"
	"github.com/get-glu/glu/pkg/triggers/schedule"
	"github.com/get-glu/glu/ui"
	"github.com/opencontainers/go-digest"
	"gopkg.in/yaml.v3"
)

func run(ctx context.Context) error {
	system := glu.NewSystem(ctx, glu.Name("mypipelines"), glu.WithUI(ui.FS()))
	config, err := system.Configuration()
	if err != nil {
		return err
	}

	if err := pipelines.NewBuilder(ctx, config, glu.Name("checkout"), NewCheckoutResource).
		NewPhase(func(b pipelines.Builder[*CheckoutResource]) (edges.Phase[*CheckoutResource], error) {
			// fetch the configured OCI repositority source named "checkout"
			return pipelines.OCIPhase(b, glu.Name("oci"), "checkout")
		}).
		PromotesTo(func(b pipelines.Builder[*CheckoutResource]) (edges.UpdatablePhase[*CheckoutResource], error) {
			// build a phase for the staging environment which source from the git repository
			// configure it to promote from the OCI phase
			return pipelines.GitPhase(b, glu.Name("staging", glu.Label("env", "staging")), "checkout",
				git.ProposeChanges[*CheckoutResource](git.ProposalOption{
					Labels: []string{"automerge"},
				}))
		}, schedule.New(
			schedule.WithInterval(30*time.Second),
		)).
		PromotesTo(func(b pipelines.Builder[*CheckoutResource]) (edges.UpdatablePhase[*CheckoutResource], error) {
			// build a phase for the production environment which source from the git repository
			// configure it to promote from the staging git phase
			return pipelines.GitPhase(b, glu.Name("production", glu.Label("env", "production")), "checkout")
		}).
		Build(system); err != nil {
		return err
	}

	return system.Run()
}

// CheckoutResource is a custom envelope for carrying our specific repository configuration
// from one source to the next in our pipeline.
type CheckoutResource struct {
	oci.BaseResource
}

// NewCheckoutResource constructs a new instance of the CheckoutResource.
// This function is required for creating a new pipeline.
func NewCheckoutResource() *CheckoutResource {
	return &CheckoutResource{}
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

// ReadFrom is a Git specific resource requirement.
// It specifies how to read the resource from a target Filesystem.
// The type should navigate and source the relevant state from the fileystem provided.
// The function is also provided with metadata for the calling phase.
// This allows the defining type to adjust behaviour based on the context of the phase.
// Here we are reading a yaml file from a directory signified by a label ("env") on the phase metadata.
func (c *CheckoutResource) ReadFrom(_ context.Context, phase glu.Descriptor, fs fs.Filesystem) error {
	fi, err := fs.OpenFile(
		fmt.Sprintf("/env/%s/apps/checkout/deployment.yaml", phase.Metadata.Name),
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

	c.ImageDigest = digest.Digest(manifest.Digest)

	return nil
}

// WriteTo is a Git specific resource requirement.
// It specifies how to write the resource to a target Filesystem.
// The type should navigate and encode the state of the resource to the target Filesystem.
// The function is also provided with metadata for the calling phase.
// This allows the defining type to adjust behaviour based on the context of the phase.
// Here we are writing a yaml file to a directory signified by a label ("env") on the phase metadata.
func (c *CheckoutResource) WriteTo(ctx context.Context, phase glu.Descriptor, fs fs.Filesystem) error {
	fi, err := fs.OpenFile(
		fmt.Sprintf("/env/%s/apps/checkout/deployment.yaml", phase.Metadata.Name),
		os.O_RDONLY,
		0644,
	)
	if err != nil {
		return err
	}

	defer fi.Close()

	return yaml.NewEncoder(fi).Encode(c)
}
