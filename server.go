package glu

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/get-glu/glu/pkg/core"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	system *System
	router *chi.Mux
	ui     fs.FS
}

func newServer(system *System, ui fs.FS) *Server {
	s := &Server{
		system: system,
		router: chi.NewRouter(),
		ui:     ui,
	}

	s.setupRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	s.router.Mount("/debug", middleware.Profiler())
	s.router.Mount("/metrics", promhttp.Handler())

	if s.ui != nil {
		s.router.Mount("/", http.FileServer(http.FS(s.ui)))
	}

	s.router.Group(func(r chi.Router) {
		r.Use(middleware.StripSlashes)
		// TODO: make CORS configurable
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"http://*", "https://*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: false,
			MaxAge:           300,
		}))
		r.Use(middleware.SetHeader("Content-Type", "application/json"))

		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// API routes
		r.Route("/api/v1", func(r chi.Router) {
			r.Get("/", s.getRoot)
			r.Get("/pipelines", s.listPipelines)
			r.Get("/pipelines/{pipeline}", s.getPipeline)
			r.Get("/pipelines/{pipeline}/phases/{phase}", s.getPhase)
			r.Get("/pipelines/{pipeline}/phases/{phase}/history", s.phaseHistory)
			r.Post("/pipelines/{pipeline}/phases/{phase}/promote", s.promotePhase)
		})
	})
}

func (s *Server) getRoot(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(s.system.meta); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type listPipelinesResponse struct {
	// TODO: does a system have metadata?
	//	Metadata  Metadata           `json:"metadata"`
	Pipelines []pipelineResponse `json:"pipelines"`
}

type pipelineResponse struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
	Phases []phaseResponse   `json:"phases,omitempty"`
}

type phaseResponse struct {
	Metadata  core.Metadata    `json:"metadata,omitempty"`
	DependsOn string           `json:"depends_on,omitempty"`
	Source    core.Metadata    `json:"source,omitempty"`
	Resource  resourceResponse `json:"resource,omitempty"`
}

type resourceResponse struct {
	Synced      bool              `json:"synced,omitempty"`
	Digest      string            `json:"digest,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

func (s *Server) createPhaseResponse(ctx context.Context, phase core.Phase, dependencies map[core.Phase]core.Phase) (phaseResponse, error) {
	var dependsOn string

	if dependencies != nil {
		if d, ok := dependencies[phase]; ok && d != nil {
			dependsOn = d.Metadata().Name
		}
	}

	v, err := phase.Get(ctx)
	if err != nil {
		return phaseResponse{}, err
	}

	var annotations map[string]string
	if r, ok := v.(core.ResourceWithAnnotations); ok {
		annotations = r.Annotations()
	}

	digest, err := v.Digest()
	if err != nil {
		return phaseResponse{}, err
	}

	synced, err := phase.Synced(ctx)
	if err != nil {
		return phaseResponse{}, err
	}

	return phaseResponse{
		Metadata:  phase.Metadata(),
		DependsOn: dependsOn,
		Source:    phase.Source(),
		Resource: resourceResponse{
			Digest:      digest,
			Annotations: annotations,
			Synced:      synced,
		},
	}, nil
}

func (s *Server) createPipelineResponse(ctx context.Context, pipeline core.Pipeline) (pipelineResponse, error) {
	dependencies := pipeline.Dependencies()
	phases := make([]phaseResponse, 0)

	for phase := range pipeline.Phases() {
		response, err := s.createPhaseResponse(ctx, phase, dependencies)
		if err != nil {
			return pipelineResponse{}, err
		}

		phases = append(phases, response)
	}

	var labels map[string]string
	if pipeline.Metadata().Labels != nil {
		labels = pipeline.Metadata().Labels
	}

	return pipelineResponse{
		Name:   pipeline.Metadata().Name,
		Labels: labels,
		Phases: phases,
	}, nil
}

// Handler methods
func (s *Server) listPipelines(w http.ResponseWriter, r *http.Request) {
	var (
		ctx               = r.Context()
		pipelineResponses = make([]pipelineResponse, 0, len(s.system.pipelines))
	)

	for _, pipeline := range s.system.pipelines {
		response, err := s.createPipelineResponse(ctx, pipeline)
		if err != nil {
			slog.Error("building pipeline response", "path", r.URL.Path, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pipelineResponses = append(pipelineResponses, response)
	}

	// TODO: handle pagination
	if err := json.NewEncoder(w).Encode(listPipelinesResponse{
		Pipelines: pipelineResponses,
	}); err != nil {
		slog.Error("encoding pipeline", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) getPipeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pipeline, err := s.system.GetPipeline(chi.URLParam(r, "pipeline"))
	if err != nil {
		slog.Debug("resource not found", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response, err := s.createPipelineResponse(ctx, pipeline)
	if err != nil {
		slog.Error("building pipeline response", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) getPhase(w http.ResponseWriter, r *http.Request) {
	pipeline, err := s.system.GetPipeline(chi.URLParam(r, "pipeline"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	phaseName := chi.URLParam(r, "phase")
	phase, err := pipeline.PhaseByName(phaseName)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, core.ErrNotFound) {
			slog.Debug("resource not found", "path", r.URL.Path, "error", err)
			status = http.StatusNotFound
		}

		http.Error(w, err.Error(), status)
		return
	}

	response, err := s.createPhaseResponse(r.Context(), phase, pipeline.Dependencies())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("encoding response", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) promotePhase(w http.ResponseWriter, r *http.Request) {
	pipeline, err := s.system.GetPipeline(chi.URLParam(r, "pipeline"))
	if err != nil {
		slog.Debug("resource not found", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	phaseName := chi.URLParam(r, "phase")
	phase, err := pipeline.PhaseByName(phaseName)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, core.ErrNotFound) {
			slog.Debug("resource not found", "path", r.URL.Path, "error", err)
			status = http.StatusNotFound
		}

		http.Error(w, err.Error(), status)
		return
	}

	result, err := phase.Promote(r.Context())
	if err != nil {
		if errors.Is(err, core.ErrNoChange) {
			slog.Debug("promotion produced no change")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		slog.Error("performing promotion", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(&result); err != nil {
		slog.Error("encoding response", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) phaseHistory(w http.ResponseWriter, r *http.Request) {
	pipeline, err := s.system.GetPipeline(chi.URLParam(r, "pipeline"))
	if err != nil {
		slog.Debug("resource not found", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	phaseName := chi.URLParam(r, "phase")
	phase, err := pipeline.PhaseByName(phaseName)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, core.ErrNotFound) {
			slog.Debug("resource not found", "path", r.URL.Path, "error", err)
			status = http.StatusNotFound
		}

		http.Error(w, err.Error(), status)
		return
	}

	history, err := phase.History(r.Context())
	if err != nil {
		slog.Error("performing promotion", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(history); err != nil {
		slog.Error("encoding response", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
