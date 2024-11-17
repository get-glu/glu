package glu

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/get-glu/glu/pkg/cli"
	"github.com/get-glu/glu/pkg/config"
	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
	"golang.org/x/sync/errgroup"
)

const defaultScheduleInternal = time.Minute

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

// System is the primary entrypoint for build a set of Glu pipelines.
// It supports functions for adding new pipelines, registering triggers
// running the API server and handly command-line inputs.
type System struct {
	ctx       context.Context
	meta      Metadata
	conf      *Config
	pipelines map[string]core.Pipeline
	schedules []Schedule
	err       error

	server *Server
}

// NewSystem constructs and configures a new system with the provided metadata.
func NewSystem(ctx context.Context, meta Metadata) *System {
	r := &System{
		ctx:       ctx,
		meta:      meta,
		pipelines: map[string]core.Pipeline{},
	}

	r.server = newServer(r)

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
func (s *System) AddPipeline(fn func(context.Context, *Config) (core.Pipeline, error)) *System {
	// skip next step if error is not nil
	if s.err != nil {
		return s
	}

	config, err := s.configuration()
	if err != nil {
		s.err = err
		return s
	}

	pipe, err := fn(s.ctx, config)
	if err != nil {
		s.err = err
		return s
	}

	s.pipelines[pipe.Metadata().Name] = pipe
	return s
}

func (s *System) configuration() (_ *Config, err error) {
	if s.conf != nil {
		return s.conf, nil
	}

	conf, err := config.ReadFromPath("glu.yaml")
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

	s.conf = newConfigSource(conf)

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
		group errgroup.Group
		srv   = http.Server{
			Addr:    ":8080", // TODO: make configurable
			Handler: s.server,
		}
	)

	group.Go(func() error {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})

	group.Go(func() error {
		slog.Info("starting server", "addr", ":8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cancel()
			return err
		}
		return nil
	})

	group.Go(func() error {
		return s.run(ctx)
	})

	return group.Wait()
}

// Schedule is a structure containing details for schedule promotion triggers.
type Schedule struct {
	interval time.Duration
	options  []containers.Option[core.PhaseOptions]
}

// SchedulePromotion creates a schedule for running automated promotion calls.
func (s *System) SchedulePromotion(opts ...containers.Option[Schedule]) *System {
	sch := Schedule{
		interval: defaultScheduleInternal,
	}

	containers.ApplyAll(&sch, opts...)

	s.schedules = append(s.schedules, sch)

	return s
}

// ScheduleInterval sets the interval on a schedule
func ScheduleInterval(d time.Duration) containers.Option[Schedule] {
	return func(s *Schedule) {
		s.interval = d
	}
}

// ScheduleMatchesPhase sets a match condition which matches a specific phase
func ScheduleMatchesPhase(c core.Phase) containers.Option[Schedule] {
	return func(s *Schedule) {
		s.options = append(s.options, core.IsPhase(c))
	}
}

// ScheduleMatchesLabel sets a match condition which matches any phase with the provided label
func ScheduleMatchesLabel(k, v string) containers.Option[Schedule] {
	return func(s *Schedule) {
		s.options = append(s.options, core.HasLabel(k, v))
	}
}

func (s *System) run(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, sch := range s.schedules {
		wg.Add(1)
		go func(sch Schedule) {
			defer wg.Done()

			ticker := time.NewTicker(sch.interval)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					for _, pipeline := range s.pipelines {
						for phase := range pipeline.Phases(sch.options...) {
							if err := phase.Promote(ctx); err != nil {
								slog.Error("reconciling resource", "name", phase.Metadata().Name, "error", err)
							}
						}
					}

				}
			}
		}(sch)
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
