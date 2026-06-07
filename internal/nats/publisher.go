package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	natsgo "github.com/nats-io/nats.go"
	warden "github.com/nicaozx/warden-gateway"
	"go.opentelemetry.io/otel"
)

type Publisher struct {
	conn *natsgo.Conn
}

type natsHeaderCarrier struct {
	header natsgo.Header
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := natsgo.Connect(
		url,
		natsgo.ReconnectHandler(func(conn *natsgo.Conn) {
			slog.Info("reconnected to NATS", "url", conn.ConnectedUrl())
		}),
		natsgo.MaxReconnects(10),
		natsgo.ReconnectWait(5*time.Second),
		natsgo.RetryOnFailedConnect(true),
		natsgo.ConnectHandler(func(conn *natsgo.Conn) {
			slog.Info("connected to NATS", "url", conn.ConnectedUrl())
		}),
		natsgo.DisconnectErrHandler(func(conn *natsgo.Conn, err error) {
			slog.Warn("disconnected from NATS", "error", err)
		}))
	if err != nil {
		return nil, err
	}
	return &Publisher{conn: conn}, nil
}

// TODO: replace JSON with Protobuf when .proto files are introduced for gRPC
func (p *Publisher) Publish(ctx context.Context, reading warden.SensorReading) error {
	subject := fmt.Sprintf("warden.sensors.v1.%s.%s", reading.Room, reading.Type)
	readingBytes, err := json.Marshal(reading)
	if err != nil {
		return err
	}

	msg := natsgo.Msg{
		Subject: subject,
		Header:  natsgo.Header{},
		Data:    readingBytes,
	}

	otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier{msg.Header})
	return p.conn.PublishMsg(&msg)
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

func (c natsHeaderCarrier) Get(key string) string {
	return c.header.Get(key)
}

func (c natsHeaderCarrier) Set(key, value string) {
	c.header.Set(key, value)
}

func (c natsHeaderCarrier) Keys() []string {
	result := make([]string, 0, len(c.header))
	for k := range c.header {
		result = append(result, k)
	}
	return result
}
