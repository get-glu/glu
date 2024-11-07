package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/get-glu/glu"
	"github.com/get-glu/glu/pkg/controllers"
	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/fs"
	"github.com/get-glu/glu/pkg/src/git"
	"github.com/get-glu/glu/pkg/src/oci"
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
	system := glu.NewSystem()
	config, err := system.Configuration()
	if err != nil {
		return err
	}

	ociSource, err := oci.New[*CheckoutResource]("checkout", config)
	if err != nil {
		return err
	}

	gitSource, err := git.NewSource[*CheckoutResource](ctx, "checkout", config, git.ProposeChanges, git.AutoMerge)
	if err != nil {
		return err
	}

	var (
		pipeline = glu.NewPipeline(glu.Metadata{
			Name: "checkout",
		},
			NewCheckoutResource,
		)

		upstream   = glu.NewPhase("upstream")
		staging    = glu.NewPhase("staging")
		production = glu.NewPhase("production")
	)

	// oci controller
	ociController := controllers.New(pipeline, upstream, ociSource)

	// staging git controller
	gitStaging := controllers.New(pipeline, staging, gitSource,
		// depends on oci upstream controller
		core.DependsOn(ociController))

	// force a reconcile of the staging instance every 10 seconds
	system.ScheduleReconcile(gitStaging, 10*time.Second)

	// construct and register production phase controller
	controllers.New(pipeline, production, gitSource,
		core.DependsOn(gitStaging))

	// register pipeline on system
	system.AddPipeline(pipeline)

	return system.Run(ctx)
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

func (c *CheckoutResource) ReadFrom(_ context.Context, phase *glu.Phase, fs fs.Filesystem) error {
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

	c.ImageDigest = manifest.Digest

	return nil
}

func (c *CheckoutResource) WriteTo(ctx context.Context, phase *glu.Phase, fs fs.Filesystem) error {
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
