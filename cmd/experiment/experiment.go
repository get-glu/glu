package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/get-glu/glu"
	"github.com/get-glu/glu/pkg/controllers/git"
	"github.com/get-glu/glu/pkg/controllers/oci"
	"github.com/get-glu/glu/pkg/fs"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gopkg.in/yaml.v3"
)

// func main() {
// 	ctx := context.Background()
// 	if err := run(ctx); err != nil {
// 		fmt.Fprintln(os.Stderr, err)
// 		os.Exit(1)
// 	}
// }

func run(ctx context.Context) error {
	pipeline, err := glu.NewPipeline(ctx, "myorgpipeline")
	if err != nil {
		return err
	}

	ociRepository, err := pipeline.NewOCIRepository("checkout")
	if err != nil {
		return err
	}

	gitRepository, err := pipeline.NewGitRepository("configuration")
	if err != nil {
		return err
	}

	checkoutResourceMeta := func(phase string) glu.Metadata {
		return glu.Metadata{
			Name:  "checkout",
			Phase: phase,
			Labels: map[string]string{
				"team": "ecommerce",
			},
		}
	}

	// create an OCI source for the checkout app which derives the app
	// configuration from the latest tags image digest
	checkoutResourceSource, err := oci.New(
		pipeline,
		ociRepository,
		checkoutResourceMeta("oci"),
		NewCheckoutResource)
	if err != nil {
		return err
	}

	// create a staging phase checkout app which is dependedent
	// on the OCI source
	checkoutStaging := git.New(
		pipeline,
		gitRepository,
		checkoutResourceMeta("staging"),
		NewCheckoutResource,
		// depends on the state of the OCI source reconciler
		git.DependsOn(checkoutResourceSource),
		// proposes changes and marks them to merge once
		// status checks have passed
		git.ProposeChanges,
		git.AutoMerge,
	)

	// force a reconcile of the staging instance every 10 seconds
	pipeline.ScheduleReconcile(checkoutStaging, 10*time.Second)

	// create a production phase checkout app which is dependedent
	// on the staging phase instance
	git.New(
		pipeline,
		gitRepository,
		checkoutResourceMeta("production"),
		NewCheckoutResource,
		// depends on the state of the staging reconciler
		git.DependsOn(checkoutStaging),
		// proposes changes but does not auto-merge
		git.ProposeChanges,
	)

	return glu.Run(ctx)
}

type CheckoutResource struct {
	meta glu.Metadata

	ImageDigest string `json:"digest"`
}

func NewCheckoutResource(meta glu.Metadata) *CheckoutResource {
	return &CheckoutResource{meta: meta}
}

func (c *CheckoutResource) Metadata() *glu.Metadata {
	return &c.meta
}

func (c *CheckoutResource) Digest() (string, error) {
	return c.ImageDigest, nil
}

func (c *CheckoutResource) ReadFromOCIDescriptor(d v1.Descriptor) error {
	c.ImageDigest = d.Digest.String()
	return nil
}

func (c *CheckoutResource) ReadFrom(_ context.Context, fs fs.Filesystem) error {
	fi, err := fs.OpenFile(
		fmt.Sprintf("/env/%s/apps/checkout/deployment.yaml", c.meta.Phase),
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

func (c *CheckoutResource) WriteTo(ctx context.Context, fs fs.Filesystem) error {
	fi, err := fs.OpenFile(
		fmt.Sprintf("/env/%s/apps/checkout/deployment.yaml", c.meta.Phase),
		os.O_RDONLY,
		0644,
	)
	if err != nil {
		return err
	}

	defer fi.Close()

	return yaml.NewEncoder(fi).Encode(c)
}
