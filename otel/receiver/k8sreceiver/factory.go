//go:generate mdatagen metadata.yaml
package k8sreceiver

import (
	"github.com/get-glu/glu/otel/internal/sharedcomponent"
	"github.com/get-glu/glu/otel/receiver/k8sreceiver/internal/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithLogs(createLogsConsumer, metadata.LogsStability),
	)
}

type AuthType string

const (
	AuthTypeNone           = AuthType("none")
	AuthTypeKubeConfig     = AuthType("kubeConfig")
	AuthTypeServiceAccount = AuthType("serviceAccount")
)

type Config struct {
	AuthType AuthType `mapstructure:"auth_type"`
}

func (cfg *Config) Validate() error {
	return nil
}

func createDefaultConfig() component.Config {
	return &Config{
		AuthType: AuthTypeKubeConfig,
	}
}

// This is the map of already created github receivers for particular configurations.
// We maintain this map because the Factory is asked log and metric receivers separately
// when it gets CreateLogsReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one receiver object per configuration.
var receivers = sharedcomponent.NewSharedComponents()
