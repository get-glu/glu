//go:generate mdatagen metadata.yaml
package githubreceiver

import (
	"github.com/get-glu/glu/otel/internal/sharedcomponent"
	"github.com/get-glu/glu/otel/receiver/githubreceiver/internal/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver"
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
	)
}

type Config struct {
	confighttp.ServerConfig `mapstructure:",squash"`
	Secret                  string `mapstructure:"secret"`
	ServiceNamePrefix       string `mapstructure:"service_name_prefix"`
	ServiceNameSuffix       string `mapstructure:"service_name_suffix"`
	CustomServiceName       string `mapstructure:"custom_service_name"`
}

func (cfg *Config) Validate() error {
	return nil
}

func createDefaultConfig() component.Config {
	return &Config{}
}

// This is the map of already created github receivers for particular configurations.
// We maintain this map because the Factory is asked log and metric receivers separately
// when it gets CreateLogsReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one receiver object per configuration.
var receivers = sharedcomponent.NewSharedComponents()
