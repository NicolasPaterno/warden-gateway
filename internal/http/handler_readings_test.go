package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	warden "github.com/NicolasPaterno/warden-gateway"
	httptransport "github.com/NicolasPaterno/warden-gateway/internal/http"
	"github.com/NicolasPaterno/warden-gateway/internal/service"
	"github.com/stretchr/testify/assert"
)

type mockReadingRepositoryForHandler struct {
	getReadings []warden.SensorReading
	getErr      error
}

func (m *mockReadingRepositoryForHandler) Save(_ context.Context, _ warden.SensorReading) error {
	return nil
}

func (m *mockReadingRepositoryForHandler) GetByRoomAndType(_ context.Context, _ string, _ warden.SensorType, _, _ time.Time) ([]warden.SensorReading, error) {
	return m.getReadings, m.getErr
}

func TestReadingsHandler_GetByRoomAndType(t *testing.T) {
	readings := []warden.SensorReading{
		{SensorID: "s1", Room: "bedroom", Type: warden.Temperature, Value: 22.5, Timestamp: time.Now()},
	}

	tests := []struct {
		name         string
		query        string
		repoReadings []warden.SensorReading
		repoErr      error
		wantStatus   int
	}{
		{
			name:         "returns readings as JSON",
			query:        "?room=bedroom&type=temperature&from=2026-01-01T00:00:00Z&to=2027-01-01T00:00:00Z",
			repoReadings: readings,
			repoErr:      nil,
			wantStatus:   http.StatusOK,
		},
		{
			name:       "missing room returns 400",
			query:      "?type=temperature&from=2026-01-01T00:00:00Z&to=2027-01-01T00:00:00Z",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing type returns 400",
			query:      "?room=bedroom&from=2026-01-01T00:00:00Z&to=2027-01-01T00:00:00Z",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid from returns 400",
			query:      "?room=bedroom&type=temperature&from=invalid&to=2027-01-01T00:00:00Z",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid to returns 400",
			query:      "?room=bedroom&type=temperature&from=2026-01-01T00:00:00Z&to=invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service error returns 500",
			query:      "?room=bedroom&type=temperature&from=2026-01-01T00:00:00Z&to=2027-01-01T00:00:00Z",
			repoErr:    errors.New("db error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewReadingService(&mockReadingRepositoryForHandler{
				getReadings: tt.repoReadings,
				getErr:      tt.repoErr,
			})
			handler := httptransport.NewReadingsHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/api/readings"+tt.query, nil)
			w := httptest.NewRecorder()
			handler.GetByRoomAndType(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusOK {
				var result []warden.SensorReading
				assert.NoError(t, json.NewDecoder(w.Body).Decode(&result))
				assert.Len(t, result, len(tt.repoReadings))
			}
		})
	}
}
