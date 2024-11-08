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

func run(ctx context.Context) error {
	return glu.NewSystem().AddPipeline(func(config glu.Config, sch glu.Scheduler) (glu.Pipeline, error) {
		ociSource, err := oci.New[*CheckoutResource]("checkout", config)
		if err != nil {
			return nil, err
		}

		gitSource, err := git.NewSource[*CheckoutResource](ctx, "checkout", config, git.ProposeChanges, git.AutoMerge)
		if err != nil {
			return nil, err
		}

		// create initial (empty) pipeline
		pipeline := glu.NewPipeline(glu.Name("checkout"), NewCheckoutResource)

		// build a controller which sources from the OCI repository
		ociController := controllers.New(glu.Name("oci"), pipeline, ociSource)

		// build a controller for the staging environment which source from the git repository
		// configure it to promote from the OCI controller
		gitStaging := controllers.New(glu.Metadata{
			Name:   "git-staging",
			Labels: map[string]string{"env": "staging"},
		}, pipeline, gitSource, core.PromotesFrom(ociController))

		// build a controller for the production environment which source from the git repository
		// configure it to promote from the staging git controller
		_ = controllers.New(core.Metadata{
			Name:   "git-production",
			Labels: map[string]string{"env": "production"},
		}, pipeline, gitSource, core.PromotesFrom(gitStaging))

		// schedule a reconcile of any controllers with the label pair env=staging
		sch.ScheduleReconcile(
			glu.ScheduleInterval(10*time.Second),
			glu.ScheduleMatchesLabel("env", "staging"),
			// alternatively, the controller instance can be target directly with:
			// glu.ScheduleMatchesController(gitStaging),
		)

		// return configured pipeline to the system
		return pipeline, nil
	}).Run(ctx)
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
