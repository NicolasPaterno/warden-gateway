package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	_ "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewRouter(wsHandler *WsHandler, readingsHandler *ReadingsHandler, healthHandler *HealthHandler) http.Handler {
	r := chi.NewRouter()

	//API
	r.Handle("/ws", wsHandler)
	r.Get("/api/readings", readingsHandler.GetByRoomAndType)

	//server health
	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)

	//observability
	r.Handle("/metrics", promhttp.Handler())

	return otelhttp.NewHandler(r, "warden-gateway")
}
