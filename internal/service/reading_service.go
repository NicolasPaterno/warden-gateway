package service

import (
	"context"
	"time"

	"github.com/NicolasPaterno/warden-gateway"
)

type ReadingService struct {
	repo warden.ReadingRepository
}

func NewReadingService(repo warden.ReadingRepository) *ReadingService {
	return &ReadingService{repo: repo}
}

func (s *ReadingService) Save(ctx context.Context, reading warden.SensorReading) error {
	return s.repo.Save(ctx, reading)
}

func (s *ReadingService) GetByRoomAndType(ctx context.Context, tenantID, room string, sensorType warden.SensorType, from, to time.Time) ([]warden.SensorReading, error) {
	return s.repo.GetByRoomAndType(ctx, tenantID, room, sensorType, from, to)
}

func (s *ReadingService) ListRooms(ctx context.Context, tenantID string) ([]string, error) {
	return s.repo.ListRooms(ctx, tenantID)
}
