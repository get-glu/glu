package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/ui"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	system *core.System
	router *chi.Mux
	ui     fs.FS
}

func New(system *core.System) *Server {
	s := &Server{
		system: system,
		router: chi.NewRouter(),
		ui:     ui.FS(),
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
			r.Get("/pipelines", s.listPipelines)
			r.Get("/pipelines/{pipeline}", s.getPipeline)
			r.Get("/pipelines/{pipeline}/phases/{phase}", s.getPhase)
		})
	})
}

type listPipelinesResponse struct {
	Pipelines []pipelineResponse `json:"pipelines"`
}

type pipelineResponse struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
	Phases []phaseResponse   `json:"phases,omitempty"`
	Edges  []edgeResponse    `json:"edges,omitempty"`
}

type phaseResponse struct {
	Descriptor core.Descriptor `json:"descriptor,omitempty"`
}

type edgeResponse struct {
	Kind string          `json:"kind,omitempty"`
	From core.Descriptor `json:"from,omitempty"`
	To   core.Descriptor `json:"to,omitempty"`
}

func (s *Server) createPhaseResponse(_ context.Context, phase core.Phase) (phaseResponse, error) {
	return phaseResponse{
		Descriptor: phase.Descriptor(),
	}, nil
}

func (s *Server) createPipelineResponse(ctx context.Context, pipeline *core.Pipeline) (pipelineResponse, error) {
	var labels map[string]string
	if pipeline.Metadata().Labels != nil {
		labels = pipeline.Metadata().Labels
	}

	phases := make([]phaseResponse, 0)
	for phase := range pipeline.Phases() {
		response, err := s.createPhaseResponse(ctx, phase)
		if err != nil {
			return pipelineResponse{}, err
		}

		phases = append(phases, response)
	}

	// Sort phases by name for stability
	slices.SortFunc(phases, func(a, b phaseResponse) int {
		return strings.Compare(strings.ToLower(a.Descriptor.Metadata.Name), strings.ToLower(b.Descriptor.Metadata.Name))
	})

	edges := make([]edgeResponse, 0)
	for _, outgoing := range pipeline.EdgesFrom() {
		for _, edge := range outgoing {
			edges = append(edges, edgeResponse{
				From: edge.From(),
				To:   edge.To(),
			})
		}
	}

	// Sort edges by from for stability
	slices.SortFunc(edges, func(a, b edgeResponse) int {
		return strings.Compare(strings.ToLower(a.From.Metadata.Name), strings.ToLower(b.From.Metadata.Name))
	})

	return pipelineResponse{
		Name:   pipeline.Metadata().Name,
		Labels: labels,
		Phases: phases,
		Edges:  edges,
	}, nil
}

// Handler methods

func (s *Server) listPipelines(w http.ResponseWriter, r *http.Request) {
	var (
		ctx               = r.Context()
		slog              = slog.With("path", r.URL.Path)
		pipelineResponses = make([]pipelineResponse, 0)
	)

	for _, pipeline := range s.system.Pipelines() {
		response, err := s.createPipelineResponse(ctx, pipeline)
		if err != nil {
			slog.Error("building pipeline response", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pipelineResponses = append(pipelineResponses, response)
	}

	// Sort pipelines by name for stability
	slices.SortFunc(pipelineResponses, func(a, b pipelineResponse) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	// TODO: handle pagination
	if err := json.NewEncoder(w).Encode(listPipelinesResponse{
		Pipelines: pipelineResponses,
	}); err != nil {
		slog.Error("encoding pipeline", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) getPipeline(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		slog = slog.With("path", r.URL.Path)
	)

	pipeline, err := s.system.GetPipeline(chi.URLParam(r, "pipeline"))
	if err != nil {
		slog.Debug("resource not found", "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response, err := s.createPipelineResponse(ctx, pipeline)
	if err != nil {
		slog.Error("building pipeline response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) getPhase(w http.ResponseWriter, r *http.Request) {
	slog := slog.With("path", r.URL.Path)

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
			slog.Debug("resource not found", "error", err)
			status = http.StatusNotFound
		}

		http.Error(w, err.Error(), status)
		return
	}

	response, err := s.createPhaseResponse(r.Context(), phase)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("encoding response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
