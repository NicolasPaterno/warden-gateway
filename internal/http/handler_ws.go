package http

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/nicaozx/warden-gateway/internal/hub"
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
		log.Println(err)
		return
	}
	h.hub.ServeWs(conn)
}
