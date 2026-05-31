package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	_ "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(handler *WsHandler) http.Handler {
	r := chi.NewRouter()

	r.Handle("/ws", handler)

	return r
}
