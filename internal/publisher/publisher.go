package publisher

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nicaozx/warden-gateway/internal/sensor"
)

type Publisher struct {
	conn *nats.Conn
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &Publisher{conn: conn}, nil
}

// TODO: replace JSON with Protobuf when .proto files are introduced for gRPC
func (p *Publisher) Publish(reading sensor.Reading) error {
	subject := fmt.Sprintf("sensors.%s.%s", reading.Room, reading.Type)
	readingBytes, err := json.Marshal(reading)
	if err != nil {
		return err
	}
	err = p.conn.Publish(subject, readingBytes)
	if err != nil {
		return err
	}
	return nil
}
