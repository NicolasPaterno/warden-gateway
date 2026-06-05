package http

import (
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicaozx/warden-gateway"
)

type HealthHandler struct {
	db   *pgxpool.Pool
	nats warden.Broker
}

func NewHealthHandler(db *pgxpool.Pool, nats warden.Broker) *HealthHandler {
	return &HealthHandler{
		db:   db,
		nats: nats,
	}
}

func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := fmt.Fprint(w, "ok")
	if err != nil {
		return
	}
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := h.nats.Ping(); err != nil {
		http.Error(w, "nats unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "ok")
}
