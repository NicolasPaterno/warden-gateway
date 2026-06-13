package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/NicolasPaterno/warden-auth/authn"
	"github.com/NicolasPaterno/warden-gateway"
	"github.com/NicolasPaterno/warden-gateway/internal/service"
)

type ReadingsHandler struct {
	svc *service.ReadingService
}

func NewReadingsHandler(svc *service.ReadingService) *ReadingsHandler {
	return &ReadingsHandler{svc: svc}
}

func (h *ReadingsHandler) GetByRoomAndType(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	claims, ok := authn.ClaimsFromContext(ctx)
	if !ok || claims.Tenant == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID := claims.Tenant

	room := r.URL.Query().Get("room")
	if room == "" {
		http.Error(w, "room is required", http.StatusBadRequest)
		return
	}

	sensorType := r.URL.Query().Get("type")
	if sensorType == "" {
		http.Error(w, "sensor type is required", http.StatusBadRequest)
		return
	}

	from, err := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	to, err := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	readings, err := h.svc.GetByRoomAndType(ctx, tenantID, room, warden.SensorType(sensorType), from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(readings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *ReadingsHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := authn.ClaimsFromContext(ctx)
	if !ok || claims.Tenant == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rooms, err := h.svc.ListRooms(ctx, claims.Tenant)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rooms == nil {
		rooms = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rooms); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
