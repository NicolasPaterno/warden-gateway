package nats

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	natsgo "github.com/nats-io/nats.go"
	warden "github.com/nicaozx/warden-gateway"
)

type Publisher struct {
	conn *natsgo.Conn
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := natsgo.Connect(
		url,
		natsgo.ReconnectHandler(func(conn *natsgo.Conn) {
			log.Printf("Reconnected to NATS: %v", conn.ConnectedUrl())
		}),
		natsgo.MaxReconnects(10),
		natsgo.ReconnectWait(5*time.Second),
		natsgo.RetryOnFailedConnect(true),
		natsgo.ConnectHandler(func(conn *natsgo.Conn) {
			log.Printf("Connected to NATS: %v", conn.ConnectedUrl())
		}),
		natsgo.DisconnectErrHandler(func(conn *natsgo.Conn, err error) {
			log.Printf("Disconnected from NATS: %v", err)
		}))
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

func (p *Publisher) Ping() error {
	if p.conn.Status() != natsgo.CONNECTED {
		return fmt.Errorf("NATS not connected: %s", p.conn.Status())
	}
	return nil
}
