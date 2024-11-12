package glu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/get-glu/glu/pkg/core"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	system *System
	router *chi.Mux
}

func newServer(system *System) *Server {
	s := &Server{
		system: system,
		router: chi.NewRouter(),
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
	s.router.Use(middleware.SetHeader("Content-Type", "application/json"))
	s.router.Use(middleware.StripSlashes)

	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// API routes
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/pipelines", s.listPipelines)
		r.Get("/pipelines/{pipeline}", s.getPipeline)
		r.Get("/pipelines/{pipeline}/controllers/{controller}", s.getController)
	})
}

type listPipelinesResponse struct {
	// TODO: does a system have metadata?
	//	Metadata  Metadata           `json:"metadata"`
	Pipelines []pipelineResponse `json:"pipelines"`
}

type pipelineResponse struct {
	Name        string               `json:"name"`
	Labels      map[string]string    `json:"labels,omitempty"`
	Controllers []controllerResponse `json:"controllers,omitempty"`
}

type controllerResponse struct {
	Name      string            `json:"name"`
	DependsOn string            `json:"depends_on,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Value     interface{}       `json:"value,omitempty"`
}

// Helper functions
func (s *Server) getPipelineByName(name string) (Pipeline, error) {
	pipeline, ok := s.system.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("pipeline not found")
	}
	return pipeline, nil
}

func (s *Server) createControllerResponse(controller core.Controller, dependencies map[core.Controller]core.Controller) controllerResponse {
	var (
		dependsOn string
		labels    map[string]string
	)

	if dependencies != nil {
		if d, ok := dependencies[controller]; ok && d != nil {
			dependsOn = d.Metadata().Name
		}
	}

	if controller.Metadata().Labels != nil {
		labels = controller.Metadata().Labels
	}

	return controllerResponse{
		Name:      controller.Metadata().Name,
		DependsOn: dependsOn,
		Labels:    labels,
	}
}

func (s *Server) createPipelineResponse(ctx context.Context, pipeline Pipeline) (pipelineResponse, error) {
	dependencies := pipeline.Dependencies()
	controllers := make([]controllerResponse, 0)

	for controller := range pipeline.Controllers() {
		response := s.createControllerResponse(controller, dependencies)

		v, err := controller.Get(ctx)
		if err != nil {
			return pipelineResponse{}, err
		}
		response.Value = v

		controllers = append(controllers, response)
	}

	var labels map[string]string
	if pipeline.Metadata().Labels != nil {
		labels = pipeline.Metadata().Labels
	}

	return pipelineResponse{
		Name:        pipeline.Metadata().Name,
		Labels:      labels,
		Controllers: controllers,
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pipelineResponses = append(pipelineResponses, response)
	}

	// TODO: handle pagination
	json.NewEncoder(w).Encode(listPipelinesResponse{
		Pipelines: pipelineResponses,
	})
}

func (s *Server) getPipeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pipeline, err := s.getPipelineByName(chi.URLParam(r, "pipeline"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response, err := s.createPipelineResponse(ctx, pipeline)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) getController(w http.ResponseWriter, r *http.Request) {
	pipeline, err := s.getPipelineByName(chi.URLParam(r, "pipeline"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	controllerName := chi.URLParam(r, "controller")
	controller, err := pipeline.ControllerByName(controllerName)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, core.ErrNotFound) {
			status = http.StatusNotFound
		}

		http.Error(w, err.Error(), status)
		return
	}

	v, err := controller.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := s.createControllerResponse(controller, pipeline.Dependencies())
	response.Value = v

	json.NewEncoder(w).Encode(response)
}
