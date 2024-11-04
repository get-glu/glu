package glu

import (
	"encoding/json"
	"net/http"

	"github.com/get-glu/glu/pkg/core"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	registry *Registry
	router   *chi.Mux
}

func newServer(registry *Registry) *Server {
	s := &Server{
		registry: registry,
		router:   chi.NewRouter(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// API routes
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/pipelines", s.listPipelines)
		r.Get("/pipelines/{pipeline}", s.getPipeline)
		r.Get("/pipelines/{pipeline}/phases/{phase}", s.getPhase)
		r.Get("/pipelines/{pipeline}/phases/{phase}/resources/{resource}", s.getResource)
	})
}

type listPipelinesResponse struct {
	// TODO: return registry metadata
	Pipelines []pipelineResponse `json:"pipelines"`
}

type phaseResponse struct {
	Name      string   `json:"name"`
	Resources []string `json:"resources"`
}

type pipelineResponse struct {
	// TODO: return pipeline metadata
	Name   string          `json:"name"`
	Phases []phaseResponse `json:"phases"`
}

type resourceResponse struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

func (s *Server) listPipelines(w http.ResponseWriter, r *http.Request) {
	var (
		pipelines         = s.registry.pipelines
		pipelineResponses = []pipelineResponse{}
	)

	for name, pipeline := range pipelines {
		phases := []phaseResponse{}
		for phase, reconcilers := range pipeline.Phases() {
			resources := []string{}
			for _, r := range reconcilers {
				resources = append(resources, r.Metadata().Name)
			}

			phases = append(phases, phaseResponse{
				Name:      phase,
				Resources: resources,
			})
		}

		pipelineResponses = append(pipelineResponses, pipelineResponse{
			Name:   name,
			Phases: phases,
		})
	}

	// TODO: handle pagination
	json.NewEncoder(w).Encode(listPipelinesResponse{
		Pipelines: pipelineResponses,
	})
}

func (s *Server) getPipeline(w http.ResponseWriter, r *http.Request) {
	pipelineName := chi.URLParam(r, "pipeline")
	pipeline, ok := s.registry.pipelines[pipelineName]
	if !ok {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	phases := []phaseResponse{}
	for phase, reconcilers := range pipeline.Phases() {
		resources := []string{}
		for _, r := range reconcilers {
			resources = append(resources, r.Metadata().Name)
		}

		phases = append(phases, phaseResponse{
			Name:      phase,
			Resources: resources,
		})
	}

	json.NewEncoder(w).Encode(pipelineResponse{
		Name:   pipelineName,
		Phases: phases,
	})
}

func (s *Server) getPhase(w http.ResponseWriter, r *http.Request) {
	var (
		pipelineName = chi.URLParam(r, "pipeline")
		phaseName    = chi.URLParam(r, "phase")
	)

	pipeline, ok := s.registry.pipelines[pipelineName]
	if !ok {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	phases := pipeline.Phases()
	reconcilers, ok := phases[phaseName]
	if !ok {
		http.Error(w, "phase not found", http.StatusNotFound)
		return
	}

	resources := []string{}
	for _, r := range reconcilers {
		resources = append(resources, r.Metadata().Name)
	}

	json.NewEncoder(w).Encode(phaseResponse{
		Name:      phaseName,
		Resources: resources,
	})
}

func (s *Server) getResource(w http.ResponseWriter, r *http.Request) {
	var (
		pipelineName = chi.URLParam(r, "pipeline")
		phaseName    = chi.URLParam(r, "phase")
		resourceName = chi.URLParam(r, "resource")
	)

	pipeline, ok := s.registry.pipelines[pipelineName]
	if !ok {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	phases := pipeline.Phases()
	reconcilers, ok := phases[phaseName]
	if !ok {
		http.Error(w, "phase not found", http.StatusNotFound)
		return
	}

	var targetResource core.Reconciler
	for _, reconciler := range reconcilers {
		if reconciler.Metadata().Name == resourceName {
			targetResource = reconciler
			break
		}
	}

	if targetResource == nil {
		http.Error(w, "resource not found", http.StatusNotFound)
		return
	}

	metadata := targetResource.Metadata()
	json.NewEncoder(w).Encode(resourceResponse{
		Name:   metadata.Name,
		Labels: metadata.Labels,
	})
}
