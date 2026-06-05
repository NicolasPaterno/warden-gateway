package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	httptransport "github.com/nicaozx/warden-gateway/internal/http"
	"github.com/nicaozx/warden-gateway/internal/hub"
)

func TestNewRouter_WSRouteExists(t *testing.T) {
	h := hub.NewHub()
	wsHandler := httptransport.NewWsHandler(h)
	router := httptransport.NewRouter(wsHandler)

	request, err := http.NewRequest("GET", "/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code == http.StatusNotFound {
		t.Errorf("ws route not found")
	}
}

func TestWsHandler_UpgradeFailsWithoutWSHeaders(t *testing.T) {
	h := hub.NewHub()
	wsHandler := httptransport.NewWsHandler(h)
	router := httptransport.NewRouter(wsHandler)

	request, err := http.NewRequest("GET", "/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Errorf("ws handler should fail with 400")
	}
}

func TestWsHandler_UpgradeSucceeds(t *testing.T) {
	h := hub.NewHub()
	wsHandler := httptransport.NewWsHandler(h)
	router := httptransport.NewRouter(wsHandler)

	response := httptest.NewServer(router)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.Run(ctx)
	conn, _, err := websocket.DefaultDialer.Dial(strings.Replace(response.URL, "http", "ws", 1)+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}
