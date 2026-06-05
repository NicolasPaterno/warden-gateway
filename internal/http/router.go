package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	_ "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(wsHandler *WsHandler, readingsHandler *ReadingsHandler, healthHandler *HealthHandler) http.Handler {
	r := chi.NewRouter()

	r.Handle("/ws", wsHandler)
	r.Get("/api/readings", readingsHandler.GetByRoomAndType)
	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)

	return r
}
