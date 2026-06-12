package pipeline

import (
	"context"
	"log/slog"
	"time"

	warden "github.com/NicolasPaterno/warden-gateway"
	"github.com/NicolasPaterno/warden-gateway/internal/metrics"
	"github.com/NicolasPaterno/warden-gateway/internal/service"
	"go.opentelemetry.io/otel"
)

// Pipeline is the single processing path for every sensor reading, regardless
// of producer: save (transactional gate) → publish (best-effort) → metrics.
type Pipeline struct {
	svc *service.ReadingService
	pub warden.Broker
}

func New(svc *service.ReadingService, pub warden.Broker) *Pipeline {
	return &Pipeline{svc: svc, pub: pub}
}

// Process runs the sequence for one reading.
//
// A save failure aborts and is returned: Postgres is the source of truth, and
// a reading that was never persisted must not reach downstream consumers.
//
// A publish failure is logged and counted but not returned: the reading is
// already persisted, and propagating the error would cause callers to retry
// and duplicate the save. A NATS outage only degrades the real-time view.
func (p *Pipeline) Process(ctx context.Context, reading warden.SensorReading) error {
	start := time.Now()

	if err := p.svc.Save(ctx, reading); err != nil {
		return err
	}
	metrics.ReadingsTotal.WithLabelValues(string(reading.Type), reading.Room).Inc()

	if err := p.pub.Publish(ctx, reading); err != nil {
		slog.Error("error on publish", "error", err)
		metrics.NATSPublishErrors.Inc()
	}

	metrics.ReadingsLatency.Observe(time.Since(start).Seconds())
	return nil
}

// Run consumes ch and feeds each reading through Process. It returns when ch
// is closed.
//
// No hub broadcast here: the hub is fed by the NATS subscriber, so readings
// reach every pod's hub regardless of which pod produced them.
func (p *Pipeline) Run(ctx context.Context, ch <-chan warden.SensorReading) {
	for reading := range ch {
		readCtx, span := otel.Tracer("warden-gateway").Start(ctx, "sensor.reading.process")
		if err := p.Process(readCtx, reading); err != nil {
			slog.Error("failed to process reading", "error", err)
		}
		span.End()
	}
}
