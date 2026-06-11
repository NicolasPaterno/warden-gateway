package warden

import "context"

type Subscriber interface {
	Subscribe(ctx context.Context, subject string, handler func(reading SensorReading)) error
	Close() error
}
