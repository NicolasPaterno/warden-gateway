package warden

import "context"

type Broker interface {
	Publish(ctx context.Context, reading SensorReading) error
	Close() error
	Ping() error
}
