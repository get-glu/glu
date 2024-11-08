package glu

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	registry *System
	router   *chi.Mux
}

func newServer(registry *System) *Server {
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
		r.Get("/pipelines/{pipeline}/controller/{controller}", s.getController)
	})
}

type listPipelinesResponse struct {
	// TODO: return registry metadata
	Pipelines []pipelineResponse `json:"pipelines"`
}

type pipelineResponse struct {
	// TODO: return pipeline metadata
	Name        string               `json:"name"`
	Controllers []controllerResponse `json:"controllers"`
}

type controllerResponse struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Value  interface{}       `json:"value"`
}

func (s *Server) listPipelines(w http.ResponseWriter, r *http.Request) {
	var (
		pipelines         = s.registry.pipelines
		pipelineResponses = []pipelineResponse{}
	)

	for name, pipeline := range pipelines {
		controllers := []controllerResponse{}
		for name, controller := range pipeline.Controllers() {
			v, err := controller.Get(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			controllers = append(controllers, controllerResponse{
				Name:  name,
				Value: v,
			})
		}

		pipelineResponses = append(pipelineResponses, pipelineResponse{
			Name:        name,
			Controllers: controllers,
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

	controllers := []controllerResponse{}
	for name, controller := range pipeline.Controllers() {
		v, err := controller.Get(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		controllers = append(controllers, controllerResponse{
			Name:   name,
			Labels: controller.Metadata().Labels,
			Value:  v,
		})
	}

	json.NewEncoder(w).Encode(pipelineResponse{
		Name:        pipelineName,
		Controllers: controllers,
	})
}

func (s *Server) getController(w http.ResponseWriter, r *http.Request) {
	var (
		pipelineName   = chi.URLParam(r, "pipeline")
		controllerName = chi.URLParam(r, "controller")
	)

	pipeline, ok := s.registry.pipelines[pipelineName]
	if !ok {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	controller, ok := pipeline.Controllers()[controllerName]
	if !ok {
		http.Error(w, "controller not found", http.StatusNotFound)
		return
	}

	v, err := controller.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(controllerResponse{
		Name:   controller.Metadata().Name,
		Labels: controller.Metadata().Labels,
		Value:  v,
	})
}
