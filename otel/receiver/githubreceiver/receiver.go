package githubreceiver

import (
	"context"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/go-github/v68/github"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type githubReceiver struct {
	cfg    *Config
	logger *zap.Logger

	tracesConsumer consumer.Traces
	server         *http.Server
	wg             sync.WaitGroup
}

func (r *githubReceiver) Start(ctx context.Context, host component.Host) error {
	r.logger.Info("Starting GitHub receiver", zap.String("endpoint", r.cfg.Endpoint))

	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
	router.Use(middleware.RequestID)

	router.Post("/", r.ServeHTTP)

	r.server = &http.Server{
		Addr:              r.cfg.Endpoint,
		Handler:           router,
		ReadHeaderTimeout: r.cfg.ServerConfig.ReadHeaderTimeout,
	}

	r.wg.Add(1)

	go func() {
		defer r.wg.Done()

		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.logger.Error("Failed to start HTTP server", zap.Error(err))
		}
	}()

	return nil
}

func (r *githubReceiver) Shutdown(ctx context.Context) error {
	r.logger.Info("Stopping GitHub receiver")

	if r.server != nil {
		if err := r.server.Shutdown(ctx); err != nil {
			r.logger.Error("Failed to shutdown HTTP server", zap.Error(err))
		}
	}

	r.wg.Wait()

	return nil
}

func (r *githubReceiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var (
		ctx    = req.Context()
		logger = r.logger.With(zap.String("path", req.URL.Path))
	)

	payload, err := github.ValidatePayload(req, []byte(r.cfg.Secret))
	if err != nil {
		logger.Debug("Payload validation failed", zap.Error(err))
		http.Error(w, "Invalid payload or signature", http.StatusBadRequest)
		return
	}

	// Determine the type of GitHub webhook event and ensure it's one we handle
	eventType := github.WebHookType(req)
	event, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		logger.Debug("Webhook parsing failed", zap.Error(err))
		http.Error(w, "Failed to parse webhook", http.StatusBadRequest)
		return
	}

	switch e := event.(type) {
	case *github.WorkflowJobEvent:
		logger.Debug("Workflow job event received", zap.Any("event", e))

		if e.GetWorkflowJob().GetStatus() != "completed" {
			logger.Debug("Skipping non-completed workflow job", zap.Any("event", e))
			w.WriteHeader(http.StatusNoContent)
			return
		}

		logger.Debug("Workflow job completed", zap.Any("event", e))
	case *github.WorkflowRunEvent:
		logger.Debug("Workflow run event received", zap.Any("event", e))

		if e.GetWorkflowRun().GetStatus() != "completed" {
			logger.Debug("Skipping non-completed workflow run", zap.Any("event", e))
			w.WriteHeader(http.StatusNoContent)
			return
		}

		logger.Debug("Workflow run completed", zap.Any("event", e))
	default:
		logger.Debug("Unhandled event type", zap.String("event_type", eventType))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	traces, err := eventToTraces(r.cfg, event)
	if err != nil {
		logger.Error("Failed to convert event to traces", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if r.tracesConsumer != nil {
		logger.Debug("Consuming traces", zap.Any("traces", traces))
		if err := r.tracesConsumer.ConsumeTraces(ctx, traces); err != nil {
			logger.Error("Failed to consume traces", zap.Error(err))
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func newReceiver(
	settings receiver.Settings,
	cfg component.Config,
) receiver.Logs {
	return &githubReceiver{
		cfg:    cfg.(*Config),
		logger: settings.Logger,
	}
}

func createTracesReceiver(
	ctx context.Context,
	settings receiver.Settings,
	ccfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	r := receivers.GetOrAdd(settings.ID, func() component.Component {
		return newReceiver(settings, ccfg.(*Config))
	})

	r.Unwrap().(*githubReceiver).tracesConsumer = nextConsumer

	return r, nil
}
