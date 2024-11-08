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
	)

	// oci controller
	ociController := controllers.New(core.Metadata{
		Name: "oci",
	}, pipeline, ociSource)

	// staging git controller
	gitStaging := controllers.New(core.Metadata{
		Name:   "git-staging",
		Labels: map[string]string{"env": "staging"},
	}, pipeline, gitSource,
		// depends on oci upstream controller
		core.DependsOn(ociController))

	// force a reconcile of the staging instance every 10 seconds
	system.ScheduleReconcile(
		glu.ScheduleInterval(10*time.Second),
		glu.ScheduleMatchesController(gitStaging),
		// alternatively, labels can be matched to reconcile all
		// controllers with a common label k/v pair:
		// glu.ScheduleMatchesLabel("env", "staging"),
	)

	// construct and register production phase controller
	controllers.New(core.Metadata{
		Name:   "git-production",
		Labels: map[string]string{"env": "production"},
	}, pipeline, gitSource,
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
