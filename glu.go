package glu

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/get-glu/glu/pkg/cli"
	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/containers"
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

// Metadata is an alias for the core Metadata structure (see core.Metadata)
type Metadata = core.Metadata

// Resource is an alias for the core Resource interface (see core.Resource)
type Resource = core.Resource

// Name is a utility for quickly creating an instance of Metadata
// with a name (required) and optional labels / annotations
func Name(name string, opts ...containers.Option[Metadata]) Metadata {
	meta := Metadata{Name: name}
	containers.ApplyAll(&meta, opts...)
	return meta
}

// Label returns a functional option for Metadata which sets
// a single label k/v pair on the provided Metadata
func Label(k, v string) containers.Option[Metadata] {
	return func(m *core.Metadata) {
		if m.Labels == nil {
			m.Labels = map[string]string{}
		}

		m.Labels[k] = v
	}
}

// Annotation returns a functional option for Metadata which sets
// a single annotation k/v pair on the provided Metadata
func Annotation(k, v string) containers.Option[Metadata] {
	return func(m *core.Metadata) {
		if m.Annotations == nil {
			m.Annotations = map[string]string{}
		}

		m.Annotations[k] = v
	}
}

type shutdownFunc func(context.Context) error

// System is the primary entrypoint for build a set of Glu pipelines.
// It supports functions for adding new pipelines, registering triggers
// running the API server and handly command-line inputs.
type System struct {
	ctx       context.Context
	meta      Metadata
	conf      *Config
	pipelines map[string]core.Pipeline
	triggers  []Trigger
	err       error

	ui            fs.FS
	server        *Server
	shutdownFuncs []shutdownFunc
}

// WithUI configures the provided fs.FS implementation to be served as the filesystem
// mounted on the root path in the API
//
// glu.NewSystem(ctx, glu.Name("mysystem"), glu.WithUI(ui.FS()))
// see: github.com/get-glu/glu/tree/main/ui sub-module for the pre-built default UI.
func WithUI(ui fs.FS) containers.Option[System] {
	return func(s *System) {
		s.ui = ui
	}
}

// NewSystem constructs and configures a new system with the provided metadata.
func NewSystem(ctx context.Context, meta Metadata, opts ...containers.Option[System]) *System {
	r := &System{
		ctx:       ctx,
		meta:      meta,
		pipelines: map[string]core.Pipeline{},
	}

	containers.ApplyAll(r, opts...)

	r.server = newServer(r, r.ui)

	return r
}

// GetPipeline returns a pipeline by name.
func (s *System) GetPipeline(name string) (core.Pipeline, error) {
	pipeline, ok := s.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("pipeline %q: %w", name, core.ErrNotFound)
	}

	return pipeline, nil
}

// Pipelines returns an iterator across all name and pipeline pairs
// previously registered on the system.
func (s *System) Pipelines() iter.Seq2[string, core.Pipeline] {
	return maps.All(s.pipelines)
}

// AddPipeline invokes a pipeline builder function provided by the caller.
// The function is provided with the systems configuration and (if successful)
// the system registers the resulting pipeline.
func (s *System) AddPipeline(pipeline core.Pipeline) *System {
	s.pipelines[pipeline.Metadata().Name] = pipeline

	return s
}

func (s *System) Configuration() (_ *Config, err error) {
	if s.conf != nil {
		return s.conf, nil
	}

	conf, err := config.ReadFromFS(os.DirFS("."))
	if err != nil {
		return nil, err
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(conf.Log.Level)); err != nil {
		return nil, err
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))

	s.conf = newConfigSource(s.ctx, conf)

	return s.conf, nil
}

// Run invokes or serves the entire system.
// Given command-line arguments are provided then the system is run as a CLI.
// Otherwise, the system runs in server mode, which means that:
// - The API is hosted on the configured port
// - Triggers are setup (schedules etc.)
func (s *System) Run() error {
	if s.err != nil {
		return s.err
	}

	ctx, cancel := signal.NotifyContext(s.ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) > 1 {
		return cli.Run(ctx, s, os.Args...)
	}

	var (
		conf  = s.conf.conf
		group *errgroup.Group
		srv   = http.Server{
			Addr:    fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port),
			Handler: s.server,
		}
	)

	s.shutdownFuncs = append(s.shutdownFuncs, srv.Shutdown)

	if conf.Metrics.Enabled {
		metricsExp, metricsShutdownFunc, err := getMetricsExporter(ctx, conf.Metrics)
		if err != nil {
			return err
		}

		s.shutdownFuncs = append(s.shutdownFuncs, metricsShutdownFunc)

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
		s.shutdownFuncs = append(s.shutdownFuncs, meterProvider.Shutdown)

		// We only want to start the runtime metrics by open telemetry if the user have chosen
		// to use OTLP because the Prometheus endpoint already exposes those metrics.
		if conf.Metrics.Exporter == config.MetricsExporterOTLP {
			err = otlpruntime.Start(otlpruntime.WithMeterProvider(meterProvider))
			if err != nil {
				return fmt.Errorf("starting runtime metric exporter: %w", err)
			}
		}
	}

	group, ctx = errgroup.WithContext(ctx)

	group.Go(func() error {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// call in reverse order to emulate pop semantics of a stack
		for _, fn := range slices.Backward(s.shutdownFuncs) {
			if err := fn(shutdownCtx); err != nil {
				slog.Error("shutting down", "error", err)
			}
		}

		return nil
	})

	var serveFunc = srv.ListenAndServe
	if conf.Server.Protocol == config.ProtocolHTTPS {
		serveFunc = func() error {
			return srv.ListenAndServeTLS(conf.Server.CertFile, conf.Server.KeyFile)
		}
	}

	group.Go(func() error {
		slog.Info("starting server", "addr", fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port))
		if err := serveFunc(); err != nil && err != http.ErrServerClosed {
			return err
		}

		slog.Debug("shutting down")
		return nil
	})

	group.Go(func() error {
		return s.runTriggers(ctx)
	})

	return group.Wait()
}

// Pipelines is a type which can list a set of configured name/Pipeline pairs.
type Pipelines interface {
	Pipelines() iter.Seq2[string, core.Pipeline]
}

// Trigger is a type with a blocking function run which can trigger
// calls to promote phases in a set of pipelines.
type Trigger interface {
	Run(context.Context, Pipelines)
}

// AddTrigger registers a Trigger to run when the system is invoked in server mode.
func (s *System) AddTrigger(trigger Trigger) *System {
	s.triggers = append(s.triggers, trigger)

	return s
}

func (s *System) runTriggers(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("running triggers: %w", err)
		}
	}()

	var wg sync.WaitGroup
	for _, trigger := range s.triggers {
		wg.Add(1)
		go func(trigger Trigger) {
			defer wg.Done()

			trigger.Run(ctx, s)
		}(trigger)
	}

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		wg.Wait()
	}()

	<-ctx.Done()

	select {
	case <-time.After(15 * time.Second):
		return errors.New("timedout waiting on shutdown of schedules")
	case <-finished:
		return ctx.Err()
	}
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
