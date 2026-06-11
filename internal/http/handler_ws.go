package http

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/NicolasPaterno/warden-auth/authn"
	"github.com/NicolasPaterno/warden-gateway/internal/hub"
	"github.com/gorilla/websocket"
)

type WsHandler struct {
	hub      *hub.Hub
	verifier *authn.Verifier
	upgrader websocket.Upgrader
}

func NewWsHandler(hub *hub.Hub, verifier *authn.Verifier) *WsHandler {
	return &WsHandler{
		hub:      hub,
		verifier: verifier,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}
func (h *WsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := wsToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	claims, err := h.verifier.Verify(r.Context(), token)
	if err != nil || claims.Tenant == "" {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	tenantID := claims.Tenant

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	h.hub.ServeWs(conn, tenantID)
}

func wsToken(r *http.Request) string {
	const prefix = "Bearer "
	if h := r.Header.Get("Authorization"); len(h) >= len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return r.URL.Query().Get("token")
}
