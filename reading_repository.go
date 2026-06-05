package warden

import (
	"context"
	"time"
)

type ReadingRepository interface {
	Save(ctx context.Context, reading SensorReading) error
	GetByRoomAndType(ctx context.Context, room string, sensorType SensorType, from, to time.Time) ([]SensorReading, error)
}
