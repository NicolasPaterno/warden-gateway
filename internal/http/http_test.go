package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	warden "github.com/nicaozx/warden-gateway"
	httptransport "github.com/nicaozx/warden-gateway/internal/http"
	"github.com/nicaozx/warden-gateway/internal/hub"
	"github.com/nicaozx/warden-gateway/internal/service"
)

type mockReadingRepository struct{}

func (m *mockReadingRepository) Save(_ context.Context, _ warden.SensorReading) error {
	return nil
}

func (m *mockReadingRepository) GetByRoomAndType(_ context.Context, _ string, _ warden.SensorType, _, _ time.Time) ([]warden.SensorReading, error) {
	return nil, nil
}

func TestNewRouter_WSRouteExists(t *testing.T) {
	h := hub.NewHub()
	wsHandler := httptransport.NewWsHandler(h)
	svc := service.NewReadingService(&mockReadingRepository{})
	readingsHandler := httptransport.NewReadingsHandler(svc)
	router := httptransport.NewRouter(wsHandler, readingsHandler)

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
	svc := service.NewReadingService(&mockReadingRepository{})
	readingsHandler := httptransport.NewReadingsHandler(svc)
	router := httptransport.NewRouter(wsHandler, readingsHandler)

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
	svc := service.NewReadingService(&mockReadingRepository{})
	readingsHandler := httptransport.NewReadingsHandler(svc)
	router := httptransport.NewRouter(wsHandler, readingsHandler)

	response := httptest.NewServer(router)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.Run(ctx)
	conn, _, err := websocket.DefaultDialer.Dial(strings.Replace(response.URL, "http", "ws", 1)+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = conn.Close()
	if err != nil {
		t.Fatal(err)
	}
}
