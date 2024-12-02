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
			r.Post("/pipelines/{pipeline}/from/{from}/to/{to}/perform", s.edgePerform)
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
	Edges  []edgeResponse    `json:"edges,omitempty"`
}

type phaseResponse struct {
	Descriptor core.Descriptor  `json:"descriptor,omitempty"`
	Resource   resourceResponse `json:"resource,omitempty"`
}

type edgeResponse struct {
	Kind       string          `json:"kind,omitempty"`
	From       core.Descriptor `json:"from,omitempty"`
	To         core.Descriptor `json:"to,omitempty"`
	CanPerform bool            `json:"can_perform,omitempty"`
}

type resourceResponse struct {
	Digest      string            `json:"digest,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

func (s *Server) createPhaseResponse(ctx context.Context, phase core.Phase) (phaseResponse, error) {
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

	return phaseResponse{
		Descriptor: phase.Descriptor(),
		Resource: resourceResponse{
			Digest:      digest,
			Annotations: annotations,
		},
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

	edges := make([]edgeResponse, 0)
	for _, outgoing := range pipeline.Edges() {
		for _, edge := range outgoing {
			canPerform, err := edge.CanPerform(ctx)
			if err != nil {
				return pipelineResponse{}, err
			}

			edges = append(edges, edgeResponse{
				Kind:       edge.Kind(),
				From:       edge.From(),
				To:         edge.To(),
				CanPerform: canPerform,
			})
		}
	}

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

	response, err := s.createPhaseResponse(r.Context(), phase)
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

func (s *Server) edgePerform(w http.ResponseWriter, r *http.Request) {
	pipeline, err := s.system.GetPipeline(chi.URLParam(r, "pipeline"))
	if err != nil {
		slog.Debug("resource not found", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var (
		from = chi.URLParam(r, "from")
		to   = chi.URLParam(r, "to")
	)

	outgoing, ok := pipeline.Edges()[from]
	if !ok {
		slog.Debug("edge not found", "path", r.URL.Path, "error", err, "from", from)
		http.Error(w, "edge not found", http.StatusNotFound)
		return
	}

	edge, ok := outgoing[to]
	if !ok {
		slog.Debug("edge not found", "path", r.URL.Path, "error", err, "from", from, "to", to)
		http.Error(w, "edge not found", http.StatusNotFound)
		return
	}

	result, err := edge.Perform(r.Context())
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
