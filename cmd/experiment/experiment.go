package main

import (
	"context"
	"fmt"
	"os"

	"github.com/flipt-io/glu"
	"github.com/flipt-io/glu/pkg/fs"
	"github.com/flipt-io/glu/pkg/sources/oci"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gopkg.in/yaml.v3"
)

func run(ctx context.Context) error {
	var (
		pipeline = glu.NewPipeline(ctx)
		staging  = pipeline.NewPhase("staging")
		_        = pipeline.NewPhase("production")
	)

	checkoutAppSource, err := oci.New("ghcr.io/my-org/checkout:latest", func(d v1.Descriptor) (CheckoutApp, error) {
		return CheckoutApp{
			Digest: d.Digest.String(),
		}, nil
	})
	if err != nil {
		return err
	}

	_ = glu.NewInstance(
		staging,
		&CheckoutApp{},
		glu.DerivedFrom(checkoutAppSource),
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
