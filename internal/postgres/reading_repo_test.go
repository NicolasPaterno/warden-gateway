package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	warden "github.com/NicolasPaterno/warden-gateway"
	"github.com/NicolasPaterno/warden-gateway/internal/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "timescale/timescaledb:latest-pg16",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_DB":       "warden_test",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		},
		Started: true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://postgres:postgres@%s:%s/warden_test?sslmode=disable", host, port.Port())
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		CREATE EXTENSION IF NOT EXISTS timescaledb;
		CREATE TABLE IF NOT EXISTS readings (
			sensor_id   TEXT        NOT NULL,
			room        TEXT        NOT NULL,
			type        TEXT        NOT NULL,
			value       FLOAT8      NOT NULL,
			unit        TEXT        NOT NULL,
			time        TIMESTAMPTZ NOT NULL,
			tenant_id   TEXT        NOT NULL DEFAULT ''
		);
		SELECT create_hypertable('readings', 'time', if_not_exists => TRUE);
		CREATE INDEX IF NOT EXISTS idx_readings_tenant
			ON readings (tenant_id, room, type, time DESC);
	`)
	require.NoError(t, err)

	cleanup := func() {
		pool.Close()
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return pool, cleanup
}

func TestReadingRepo_Save(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := postgres.NewReadingRepo(pool)
	reading := warden.SensorReading{
		SensorID:  "s1",
		Room:      "bedroom",
		Type:      warden.Temperature,
		Value:     22.5,
		Unit:      "°C",
		Timestamp: time.Now().UTC().Truncate(time.Microsecond),
	}

	err := repo.Save(context.Background(), reading)
	assert.NoError(t, err)
}

func TestReadingRepo_GetByRoomAndType(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := postgres.NewReadingRepo(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)

	readings := []warden.SensorReading{
		{TenantID: "tenant-1", SensorID: "s1", Room: "bedroom", Type: warden.Temperature, Value: 22.5, Unit: "°C", Timestamp: now},
		{TenantID: "tenant-1", SensorID: "s1", Room: "bedroom", Type: warden.Temperature, Value: 23.0, Unit: "°C", Timestamp: now.Add(-time.Minute)},
		{TenantID: "tenant-1", SensorID: "s2", Room: "kitchen", Type: warden.Humidity, Value: 60.0, Unit: "%", Timestamp: now},
		{TenantID: "tenant-2", SensorID: "s3", Room: "bedroom", Type: warden.Temperature, Value: 99.0, Unit: "°C", Timestamp: now},
	}

	for _, r := range readings {
		require.NoError(t, repo.Save(context.Background(), r))
	}

	result, err := repo.GetByRoomAndType(
		context.Background(),
		"tenant-1",
		"bedroom",
		warden.Temperature,
		now.Add(-time.Hour),
		now.Add(time.Hour),
	)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "bedroom", result[0].Room)
	assert.Equal(t, warden.Temperature, result[0].Type)
}

func TestReadingRepo_ListRooms(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := postgres.NewReadingRepo(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)

	readings := []warden.SensorReading{
		{TenantID: "tenant-1", SensorID: "s1", Room: "kitchen", Type: warden.Temperature, Value: 22.5, Unit: "°C", Timestamp: now},
		{TenantID: "tenant-1", SensorID: "s1", Room: "bedroom", Type: warden.Temperature, Value: 23.0, Unit: "°C", Timestamp: now},
		{TenantID: "tenant-1", SensorID: "s2", Room: "bedroom", Type: warden.Humidity, Value: 60.0, Unit: "%", Timestamp: now},
		{TenantID: "tenant-2", SensorID: "s3", Room: "bathroom", Type: warden.Temperature, Value: 99.0, Unit: "°C", Timestamp: now},
	}

	for _, r := range readings {
		require.NoError(t, repo.Save(context.Background(), r))
	}

	result, err := repo.ListRooms(context.Background(), "tenant-1")

	assert.NoError(t, err)
	assert.Equal(t, []string{"bedroom", "kitchen"}, result)
}
