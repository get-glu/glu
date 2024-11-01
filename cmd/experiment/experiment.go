package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/flipt-io/glu"
	"github.com/flipt-io/glu/pkg/fs"
	"github.com/flipt-io/glu/pkg/sources/oci"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gopkg.in/yaml.v3"
)

func run(ctx context.Context) error {
	pipeline, err := glu.NewPipeline(ctx, "myorgpipeline")
	if err != nil {
		return err
	}

	repository, err := pipeline.NewRepository("configuration")
	if err != nil {
		return err
	}

	checkoutAppMeta := func(phase string) glu.Metadata {
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
	checkoutAppSource, err := oci.New(checkoutAppMeta("source"), "ghcr.io/myorg/checkout", NewCheckoutApp)
	if err != nil {
		return err
	}

	// create a staging phase checkout app which is dependedent
	// on the OCI source
	checkoutStaging := glu.NewInstance(
		pipeline,
		repository,
		checkoutAppMeta("staging"),
		NewCheckoutApp,
		glu.DependsOn(checkoutAppSource),
	)

	// force a reconcile of the staging instance every 10 seconds
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ctx.Done():
			case <-ticker.C:
				if err := checkoutStaging.Reconcile(ctx); err != nil {
					slog.Error("reconciling staging checkout app", "error", err)
					continue
				}
			}
		}
	}()

	// create a production phase checkout app which is dependedent
	// on the staging phase instance
	glu.NewInstance(
		pipeline,
		repository,
		checkoutAppMeta("production"),
		NewCheckoutApp,
		glu.DependsOn(checkoutStaging),
	)

	return pipeline.Run(ctx)
}

type CheckoutApp struct {
	meta glu.Metadata

	ImageDigest string `json:"digest"`
}

func NewCheckoutApp(meta glu.Metadata) *CheckoutApp {
	return &CheckoutApp{meta: meta}
}

func (c *CheckoutApp) Metadata() *glu.Metadata {
	return &c.meta
}

func (c *CheckoutApp) Digest() (string, error) {
	return c.ImageDigest, nil
}

func (c *CheckoutApp) ReadFromOCIDescriptor(d v1.Descriptor) error {
	c.ImageDigest = d.Digest.String()
	return nil
}

func (c *CheckoutApp) ReadFrom(_ context.Context, fs fs.Filesystem) error {
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

func (c *CheckoutApp) WriteTo(ctx context.Context, fs fs.Filesystem) error {
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
