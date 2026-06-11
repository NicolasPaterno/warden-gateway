package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	warden "github.com/NicolasPaterno/warden-gateway"
	"github.com/NicolasPaterno/warden-gateway/internal/service"
	"github.com/stretchr/testify/assert"
)

type mockReadingRepository struct {
	saveErr     error
	getReadings []warden.SensorReading
	getErr      error
}

func (m *mockReadingRepository) Save(_ context.Context, _ warden.SensorReading) error {
	return m.saveErr
}

func (m *mockReadingRepository) GetByRoomAndType(_ context.Context, _, _ string, _ warden.SensorType, _, _ time.Time) ([]warden.SensorReading, error) {
	return m.getReadings, m.getErr
}

func TestReadingService_Save(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{
			name:    "save succeeds",
			repoErr: nil,
			wantErr: false,
		},
		{
			name:    "save propagates repo error",
			repoErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewReadingService(&mockReadingRepository{saveErr: tt.repoErr})
			err := svc.Save(context.Background(), warden.SensorReading{})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReadingService_GetByRoomAndType(t *testing.T) {
	readings := []warden.SensorReading{
		{SensorID: "s1", Room: "bedroom", Type: warden.Temperature, Value: 22.5},
	}

	tests := []struct {
		name         string
		repoReadings []warden.SensorReading
		repoErr      error
		wantLen      int
		wantErr      bool
	}{
		{
			name:         "returns readings from repo",
			repoReadings: readings,
			repoErr:      nil,
			wantLen:      1,
			wantErr:      false,
		},
		{
			name:         "returns empty slice when no readings",
			repoReadings: []warden.SensorReading{},
			repoErr:      nil,
			wantLen:      0,
			wantErr:      false,
		},
		{
			name:         "propagates repo error",
			repoReadings: nil,
			repoErr:      errors.New("db error"),
			wantLen:      0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewReadingService(&mockReadingRepository{
				getReadings: tt.repoReadings,
				getErr:      tt.repoErr,
			})
			result, err := svc.GetByRoomAndType(context.Background(), "tenant-1", "bedroom", warden.Temperature, time.Now(), time.Now())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.wantLen)
			}
		})
	}
}
