package main

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/get-glu/glu/internal/parser"
	"github.com/get-glu/glu/internal/server"
	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/core"
	otlpruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil {
		slog.Error("error running glu", "error", err)
		os.Exit(1)
	}
}

type shutdownFunc func(context.Context) error

// Run invokes or serves the entire system.
// Given command-line arguments are provided then the system is run as a CLI.
// Otherwise, the system runs in server mode, which means that:
// - The API is hosted on the configured port
// - Triggers are setup (schedules etc.)
func run() error {
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

	sys, err := parser.Parse(ctx, "example.yml")
	if err != nil {
		return err
	}

	server := server.New(sys)

	var (
		srv = http.Server{
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

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// call in reverse order to emulate pop semantics of a stack
		for _, fn := range slices.Backward(shutdownFuncs) {
			if err := fn(shutdownCtx); err != nil {
				slog.Error("shutting down", "error", err)
			}
		}
	}()

	var serveFunc = srv.ListenAndServe
	if conf.Server.Protocol == config.ProtocolHTTPS {
		serveFunc = func() error {
			return srv.ListenAndServeTLS(conf.Server.CertFile, conf.Server.KeyFile)
		}
	}

	var group errgroup.Group
	group.Go(func() error {
		slog.Info("starting server", "addr", fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port))
		if err := serveFunc(); err != nil && err != http.ErrServerClosed {
			return err
		}

		slog.Debug("shutting down")
		return nil
	})

	return group.Wait()
}

// Pipelines is a type which can list a set of configured name/Pipeline pairs.
type Pipelines interface {
	Pipelines() iter.Seq2[string, *core.Pipeline]
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
			return nil, nil, err
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
