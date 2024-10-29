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
	var (
		pipeline    = glu.NewPipeline(ctx)
		staging     = pipeline.NewPhase("staging")
		production  = pipeline.NewPhase("production")
		checkoutApp = &CheckoutApp{}
	)

	// create an OCI source for the checkout app which derives the app
	// configuration from the latest tags image digest
	checkoutAppSource, err := oci.New("ghcr.io/my-org/checkout:latest", checkoutApp)
	if err != nil {
		return err
	}

	// create a staging phase checkout app which is dependedent
	// on the OCI source
	checkoutStaging := glu.NewInstance(
		staging,
		checkoutApp,
		glu.DependsOn(checkoutAppSource),
	)

	// force a reconcile of the staging instance every 10 seconds
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ctx.Done():
			case <-ticker.C:
				if _, err := checkoutStaging.Reconcile(ctx); err != nil {
					slog.Error("reconciling staging checkout app", "error", err)
					continue
				}
			}
		}
	}()

	// create a production phase checkout app which is dependedent
	// on the staging phase instance
	glu.NewInstance(
		production,
		checkoutApp,
		glu.DependsOn(checkoutStaging),
	)

	return pipeline.Run(ctx)
}

type CheckoutApp struct {
	Digest string
}

func (c *CheckoutApp) Metadata() glu.Metadata {
	return glu.Metadata{
		Name: "checkout",
		Labels: map[string]string{
			"team": "ecommerce",
		},
	}
}

func (c *CheckoutApp) ReadFromOCIDescriptor(d v1.Descriptor) error {
	c.Digest = d.Digest.String()
	return nil
}

func (c *CheckoutApp) ReadFrom(_ context.Context, phase *glu.Phase, fs fs.Filesystem) error {
	fi, err := fs.OpenFile(
		fmt.Sprintf("/env/%s/apps/checkout/deployment.yaml", phase.Name()),
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

	c.Digest = manifest.Digest

	return nil
}

func (c *CheckoutApp) WriteTo(_ context.Context, _ *glu.Phase, _ fs.Filesystem) error {
	panic("not implemented") // TODO: Implement
}
