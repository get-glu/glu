package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/get-glu/glu/internal/config"
	"github.com/get-glu/glu/internal/containers"
	"github.com/get-glu/glu/internal/parser"
	"github.com/get-glu/glu/internal/server"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	envprovider "go.opentelemetry.io/collector/confmap/provider/envprovider"
	fileprovider "go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/otelcol"
	otlpruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"golang.org/x/sync/errgroup"
)

var (
	dev          = flag.Bool("dev", false, "run in development mode")
	collectorCfg = flag.String("collector-config", "", "path to collector config file")
)

func main() {
	if err := run(); err != nil {
		slog.Error("error running glu", "error", err)
		os.Exit(1)
	}
}

type shutdownFunc func(context.Context) error

func run() error {

	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		return fmt.Errorf("missing path to glu.yml file")
	}

	if *collectorCfg == "" {
		return fmt.Errorf("missing path to collector config file")
	}

	if !strings.HasPrefix(*collectorCfg, "file:") {
		*collectorCfg = "file:" + *collectorCfg
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	shutdownFuncs := []shutdownFunc{}

	conf, err := config.ReadFromFS(os.DirFS("."))
	if err != nil {
		return err
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(conf.Log.Level)); err != nil {
		return err
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))

	sys, err := parser.Parse(ctx, args[0])
	if err != nil {
		return err
	}

	serverOpts := []containers.Option[server.Server]{}
	if !*dev {
		serverOpts = append(serverOpts, server.WithUI())
	}

	var (
		server = server.New(sys, serverOpts...)
		srv    = http.Server{
			Addr:    fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port),
			Handler: server,
		}
	)

	shutdownFuncs = append(shutdownFuncs, srv.Shutdown)

	if conf.Metrics.Enabled {
		metricsExp, metricsShutdownFunc, err := getMetricsExporter(ctx, conf.Metrics)
		if err != nil {
			return err
		}

		shutdownFuncs = append(shutdownFuncs, metricsShutdownFunc)

		metricsResource, err := resource.New(
			ctx,
			resource.WithSchemaURL(semconv.SchemaURL),
			resource.WithAttributes(
				semconv.ServiceName("glu"),
			),
			resource.WithFromEnv(),
			resource.WithTelemetrySDK(),
			resource.WithHost(),
			resource.WithProcessRuntimeVersion(),
			resource.WithProcessRuntimeName(),
		)
		if err != nil {
			return fmt.Errorf("creating metrics resource: %w", err)
		}

		meterProvider := metricsdk.NewMeterProvider(
			metricsdk.WithResource(metricsResource),
			metricsdk.WithReader(metricsExp),
		)

		otel.SetMeterProvider(meterProvider)
		shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)

		// We only want to start the runtime metrics by open telemetry if the user have chosen
		// to use OTLP because the Prometheus endpoint already exposes those metrics.
		if conf.Metrics.Exporter == config.MetricsExporterOTLP {
			err = otlpruntime.Start(otlpruntime.WithMeterProvider(meterProvider))
			if err != nil {
				return fmt.Errorf("starting runtime metric exporter: %w", err)
			}
		}
	}

	var serveFunc = srv.ListenAndServe
	if conf.Server.Protocol == config.ProtocolHTTPS {
		serveFunc = func() error {
			return srv.ListenAndServeTLS(conf.Server.CertFile, conf.Server.KeyFile)
		}
	}

	col, err := getCollector(*collectorCfg)
	if err != nil {
		return err
	}

	shutdownFuncs = append(shutdownFuncs, func(_ context.Context) error {
		slog.Info("shutting down collector")
		col.Shutdown()
		return nil
	})

	var group errgroup.Group
	group.Go(func() error {
		slog.Info("starting collector")
		err := col.Run(ctx)
		if err != nil {
			cancel()
		}
		return err
	})

	group.Go(func() error {
		slog.Info("starting server", "addr", fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port))
		if err := serveFunc(); err != nil && err != http.ErrServerClosed {
			cancel()
			return err
		}
		return nil
	})

	group.Go(func() error {
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutdownCancel()

		// call in reverse order to emulate pop semantics of a stack
		for _, fn := range slices.Backward(shutdownFuncs) {
			if err := fn(shutdownCtx); err != nil {
				slog.Error("shutting down", "error", err)
			}
		}
		return nil
	})

	return group.Wait()
}

func getCollector(configPath string) (*otelcol.Collector, error) {
	info := component.BuildInfo{
		Command:     "glu",
		Description: "Ingestion pipeline for CI/CD telemetry.",
		Version:     "",
	}

	set := otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				ProviderFactories: []confmap.ProviderFactory{
					envprovider.NewFactory(),
					fileprovider.NewFactory(),
				},
			},
		},
	}

	set.ConfigProviderSettings.ResolverSettings.URIs = []string{configPath}

	col, err := otelcol.NewCollector(set)
	if err != nil {
		return nil, fmt.Errorf("creating collector: %w", err)
	}

	return col, nil
}

func getMetricsExporter(ctx context.Context, cfg config.Metrics) (metricsdk.Reader, shutdownFunc, error) {
	var (
		metricExp          metricsdk.Reader
		metricShutdownFunc shutdownFunc = func(context.Context) error { return nil }
		err                error
	)

	switch cfg.Exporter {
	case config.MetricsExporterPrometheus:
		// exporter registers itself on the prom client DefaultRegistrar
		metricExp, err = prometheus.New()
		if err != nil {
			return nil, nil, fmt.Errorf("creating prometheus metrics exporter: %w", err)
		}

	case config.MetricsExporterOTLP:
		u, err := url.Parse(cfg.OTLP.Endpoint)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing otlp endpoint: %w", err)
		}

		var exporter metricsdk.Exporter

		switch u.Scheme {
		case "https":
			exporter, err = otlpmetrichttp.New(ctx,
				otlpmetrichttp.WithEndpoint(u.Host+u.Path),
				otlpmetrichttp.WithHeaders(cfg.OTLP.Headers),
			)
			if err != nil {
				return nil, nil, fmt.Errorf("creating otlp metrics exporter: %w", err)
			}
		case "http":
			exporter, err = otlpmetrichttp.New(ctx,
				otlpmetrichttp.WithEndpoint(u.Host+u.Path),
				otlpmetrichttp.WithHeaders(cfg.OTLP.Headers),
				otlpmetrichttp.WithInsecure(),
			)
			if err != nil {
				return nil, nil, fmt.Errorf("creating otlp metrics exporter: %w", err)
			}
		default:
			return nil, nil, fmt.Errorf("unsupported metrics exporter scheme: %s", u.Scheme)
		}

		metricExp = metricsdk.NewPeriodicReader(exporter)
		metricShutdownFunc = func(ctx context.Context) error {
			return exporter.Shutdown(ctx)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported metrics exporter: %s", cfg.Exporter)
	}

	return metricExp, metricShutdownFunc, err
}
