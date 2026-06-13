package pipeline_test

import (
	"context"
	"errors"
	"testing"
	"time"

	warden "github.com/NicolasPaterno/warden-gateway"
	"github.com/NicolasPaterno/warden-gateway/internal/metrics"
	"github.com/NicolasPaterno/warden-gateway/internal/pipeline"
	"github.com/NicolasPaterno/warden-gateway/internal/service"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func init() {
	_ = metrics.Register()
}

type mockReadingRepository struct {
	saveErr   error
	saveCalls int
}

func (m *mockReadingRepository) Save(_ context.Context, _ warden.SensorReading) error {
	m.saveCalls++
	return m.saveErr
}

func (m *mockReadingRepository) GetByRoomAndType(_ context.Context, _, _ string, _ warden.SensorType, _, _ time.Time) ([]warden.SensorReading, error) {
	return nil, nil
}

func (m *mockReadingRepository) ListRooms(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

type mockBroker struct {
	publishErr   error
	publishCalls int
	lastReading  warden.SensorReading
}

func (m *mockBroker) Publish(_ context.Context, reading warden.SensorReading) error {
	m.publishCalls++
	m.lastReading = reading
	return m.publishErr
}

func (m *mockBroker) Close() error { return nil }
func (m *mockBroker) Ping() error  { return nil }

func newPipeline(repo *mockReadingRepository, broker *mockBroker) *pipeline.Pipeline {
	return pipeline.New(service.NewReadingService(repo), broker)
}

func testReading() warden.SensorReading {
	return warden.SensorReading{
		TenantID:  "tenant-a",
		SensorID:  "s1",
		Room:      "bedroom",
		Type:      warden.Temperature,
		Value:     22.5,
		Unit:      "°C",
		Timestamp: time.Now(),
	}
}

func TestPipeline_Process(t *testing.T) {
	tests := []struct {
		name             string
		saveErr          error
		publishErr       error
		wantErr          bool
		wantPublishCalls int
	}{
		{
			name:             "save and publish succeed",
			saveErr:          nil,
			publishErr:       nil,
			wantErr:          false,
			wantPublishCalls: 1,
		},
		{
			name:             "save failure aborts and skips publish",
			saveErr:          errors.New("db error"),
			publishErr:       nil,
			wantErr:          true,
			wantPublishCalls: 0,
		},
		{
			name:             "publish failure is swallowed after successful save",
			saveErr:          nil,
			publishErr:       errors.New("nats down"),
			wantErr:          false,
			wantPublishCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockReadingRepository{saveErr: tt.saveErr}
			broker := &mockBroker{publishErr: tt.publishErr}
			p := newPipeline(repo, broker)

			err := p.Process(context.Background(), testReading())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, 1, repo.saveCalls)
			assert.Equal(t, tt.wantPublishCalls, broker.publishCalls)
		})
	}
}

func TestPipeline_Process_PassesReadingToBroker(t *testing.T) {
	repo := &mockReadingRepository{}
	broker := &mockBroker{}
	p := newPipeline(repo, broker)
	reading := testReading()

	err := p.Process(context.Background(), reading)

	assert.NoError(t, err)
	assert.Equal(t, reading, broker.lastReading)
}

func TestPipeline_Process_Metrics(t *testing.T) {
	t.Run("save failure does not increment readings total", func(t *testing.T) {
		counter := metrics.ReadingsTotal.WithLabelValues(string(warden.Temperature), "bedroom")
		before := testutil.ToFloat64(counter)

		p := newPipeline(&mockReadingRepository{saveErr: errors.New("db error")}, &mockBroker{})
		_ = p.Process(context.Background(), testReading())

		assert.Equal(t, before, testutil.ToFloat64(counter))
	})

	t.Run("successful save increments readings total", func(t *testing.T) {
		counter := metrics.ReadingsTotal.WithLabelValues(string(warden.Temperature), "bedroom")
		before := testutil.ToFloat64(counter)

		p := newPipeline(&mockReadingRepository{}, &mockBroker{})
		err := p.Process(context.Background(), testReading())

		assert.NoError(t, err)
		assert.Equal(t, before+1, testutil.ToFloat64(counter))
	})

	t.Run("publish failure increments nats error counter", func(t *testing.T) {
		before := testutil.ToFloat64(metrics.NATSPublishErrors)

		p := newPipeline(&mockReadingRepository{}, &mockBroker{publishErr: errors.New("nats down")})
		err := p.Process(context.Background(), testReading())

		assert.NoError(t, err)
		assert.Equal(t, before+1, testutil.ToFloat64(metrics.NATSPublishErrors))
	})
}

func TestPipeline_Run(t *testing.T) {
	t.Run("processes every reading until channel closes", func(t *testing.T) {
		repo := &mockReadingRepository{}
		broker := &mockBroker{}
		p := newPipeline(repo, broker)

		ch := make(chan warden.SensorReading, 3)
		ch <- testReading()
		ch <- testReading()
		ch <- testReading()
		close(ch)

		p.Run(context.Background(), ch)

		assert.Equal(t, 3, repo.saveCalls)
		assert.Equal(t, 3, broker.publishCalls)
	})

	t.Run("continues after a process error", func(t *testing.T) {
		repo := &mockReadingRepository{saveErr: errors.New("db error")}
		broker := &mockBroker{}
		p := newPipeline(repo, broker)

		ch := make(chan warden.SensorReading, 2)
		ch <- testReading()
		ch <- testReading()
		close(ch)

		p.Run(context.Background(), ch)

		assert.Equal(t, 2, repo.saveCalls)
		assert.Equal(t, 0, broker.publishCalls)
	})
}
