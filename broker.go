package warden

type Broker interface {
	Publish(reading SensorReading) error
	Close() error
	Ping() error
}
