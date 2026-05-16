package nats

import (
	"encoding/json"
	"fmt"

	natsgo "github.com/nats-io/nats.go"
	warden "github.com/nicaozx/warden-gateway"
)

type Publisher struct {
	conn *natsgo.Conn
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := natsgo.Connect(url)
	if err != nil {
		return nil, err
	}
	return &Publisher{conn: conn}, nil
}

// TODO: replace JSON with Protobuf when .proto files are introduced for gRPC
func (p *Publisher) Publish(reading warden.SensorReading) error {
	subject := fmt.Sprintf("warden.sensors.v1.%s.%s", reading.Room, reading.Type)
	readingBytes, err := json.Marshal(reading)
	if err != nil {
		return err
	}
	return p.conn.Publish(subject, readingBytes)
}

func (p *Publisher) Close() error {
	return p.conn.Drain()
}
