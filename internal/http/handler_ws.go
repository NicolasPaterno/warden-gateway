package http

import (
	"log/slog"
	"net/http"

	"github.com/NicolasPaterno/warden-gateway/internal/hub"
	"github.com/gorilla/websocket"
)

type WsHandler struct {
	hub      *hub.Hub
	upgrader websocket.Upgrader
}

func NewWsHandler(hub *hub.Hub) *WsHandler {
	return &WsHandler{
		hub: hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}
func (h *WsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	h.hub.ServeWs(conn)
}
