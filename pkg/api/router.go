package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

func Router(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(RequestLogger(h.log))

	// --- Health check ---
	r.Get("/healthz", h.Health)

	// --- Swagger UI ---
	// Serves:
	//   /swagger/index.html
	//   /swagger/doc.json
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// --- API v1 ---
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/gpus", h.ListGPUs)
		r.Get("/gpus/{id}/telemetry", h.GetTelemetry)
	})

	return r
}
