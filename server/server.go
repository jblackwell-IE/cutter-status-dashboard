package server

import (
	"context"
	"net/http"

	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/IdeaEvolver/cutter-pkg/service"
	"github.com/IdeaEvolver/cutter-status-dashboard/database"
	"github.com/IdeaEvolver/cutter-status-dashboard/healthchecks"
	"github.com/IdeaEvolver/cutter-status-dashboard/metrics"
	"github.com/IdeaEvolver/cutter-status-dashboard/status"
	"github.com/go-chi/chi"
	"github.com/rs/cors"
	"go.opencensus.io/plugin/ochttp"
)

type StatusStore interface {
	UpdateStatus(ctx context.Context, service, status string) error
	GetAllStatuses(ctx context.Context) ([]*status.AllStatuses, error)
	GetStatus(ctx context.Context, service string) (*status.Status, error)
}

type Handler struct {
	Statuses       StatusStore
	Healthchecks   *healthchecks.Client
	Metrics        *metrics.Metrics
	DatabaseHealth *database.Health
}

func New(cfg *service.Config, handler *Handler) *service.Server {
	router := chi.NewRouter()

	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{http.MethodHead, http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodPost},
		ExposedHeaders:   []string{"Authorization"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler)

	router.Handle("/*", http.FileServer(http.Dir("./dist")))
	router.HandleFunc("/healthcheck", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		return
	})

	router.Route("/api/v1", func(router chi.Router) {
		router.Route("/", func(router chi.Router) {
			router.Method("GET", "/get-all-statuses", service.JsonHandler(handler.GetAllStatuses))
			router.Method("GET", "/get-status", service.JsonHandler(handler.GetStatus))
		})
	})

	httpHandler := &ochttp.Handler{
		// Use the Google Cloud propagation format.
		Propagation: &propagation.HTTPFormat{},
		Handler:     router,
	}

	return service.GracefulServer(cfg, httpHandler)
}
