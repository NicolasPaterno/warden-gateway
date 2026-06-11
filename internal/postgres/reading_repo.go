package postgres

import (
	"context"
	"time"

	"github.com/NicolasPaterno/warden-gateway"
	"github.com/NicolasPaterno/warden-gateway/db/generated"
	"github.com/jackc/pgx/v5/pgtype"
)

type ReadingRepo struct {
	queries *generated.Queries
}

var _ warden.ReadingRepository = (*ReadingRepo)(nil)

func NewReadingRepo(db generated.DBTX) *ReadingRepo {
	return &ReadingRepo{
		queries: generated.New(db),
	}
}

func (r *ReadingRepo) Save(ctx context.Context, reading warden.SensorReading) error {
	params := generated.InsertReadingParams{
		TenantID: reading.TenantID,
		SensorID: reading.SensorID,
		Room:     reading.Room,
		Type:     string(reading.Type),
		Value:    reading.Value,
		Unit:     reading.Unit,
		Time:     pgtype.Timestamptz{Time: reading.Timestamp, Valid: true},
	}
	return r.queries.InsertReading(ctx, params)
}

func (r *ReadingRepo) GetByRoomAndType(ctx context.Context, tenantID, room string, sensorType warden.SensorType, from, to time.Time) ([]warden.SensorReading, error) {
	params := generated.GetReadingsByRoomAndTypeParams{
		TenantID: tenantID,
		Room:     room,
		Type:     string(sensorType),
		Time:     pgtype.Timestamptz{Time: from, Valid: true},
		Time_2:   pgtype.Timestamptz{Time: to, Valid: true},
	}
	response, err := r.queries.GetReadingsByRoomAndType(ctx, params)
	if err != nil {
		return nil, err
	}

	results := make([]warden.SensorReading, 0, len(response))

	for _, row := range response {
		results = append(results, warden.SensorReading{
			TenantID:  row.TenantID,
			SensorID:  row.SensorID,
			Room:      row.Room,
			Type:      warden.SensorType(row.Type),
			Value:     row.Value,
			Unit:      row.Unit,
			Timestamp: row.Time.Time,
		})
	}
	return results, nil
}
