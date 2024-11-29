package config

import "fmt"

type Metrics struct {
	Enabled  bool            `glu:"enabled"`
	Exporter MetricsExporter `glu:"exporter"`
	OTLP     *MetricsOTLP    `glu:"otlp"`
}

type MetricsExporter string

const (
	MetricsExporterPrometheus MetricsExporter = "prometheus"
	MetricsExporterOTLP       MetricsExporter = "otlp"
)

type MetricsOTLP struct {
	Endpoint string            `glu:"endpoint"`
	Headers  map[string]string `glu:"headers"`
}

func (c *Metrics) setDefaults() error {
	c.Enabled = true

	if c.Exporter == "" {
		c.Exporter = MetricsExporterPrometheus
	}

	return nil
}

func (c *Metrics) validate() error {
	switch c.Exporter {
	case MetricsExporterPrometheus:
	case MetricsExporterOTLP:
		return c.OTLP.validate()
	default:
		return fmt.Errorf("unexpected metrics exporter %q", c.Exporter)
	}

	return nil
}

func (c *MetricsOTLP) validate() error {
	if c.Endpoint == "" {
		return errFieldRequired("endpoint")
	}

	return nil
}
