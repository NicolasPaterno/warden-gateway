package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nicaozx/warden-gateway"
	"github.com/nicaozx/warden-gateway/internal/service"
)

type ReadingsHandler struct {
	svc *service.ReadingService
}

func NewReadingsHandler(svc *service.ReadingService) *ReadingsHandler {
	return &ReadingsHandler{svc: svc}
}

func (h *ReadingsHandler) GetByRoomAndType(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
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

	readings, err := h.svc.GetByRoomAndType(ctx, room, warden.SensorType(sensorType), from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readings)
}
