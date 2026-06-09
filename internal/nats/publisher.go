package nats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	warden "github.com/NicolasPaterno/warden-gateway"
	sensorv1 "github.com/NicolasPaterno/warden-proto/gen/go/warden/sensor/v1"
	natsgo "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (p *Publisher) Publish(ctx context.Context, reading warden.SensorReading) error {
	subject := fmt.Sprintf("warden.sensors.v1.%s.%s", reading.Room, reading.Type)
	protoReading := toProtoReading(reading)
	readingBytes, err := proto.Marshal(protoReading)
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

func toProtoReading(r warden.SensorReading) *sensorv1.SensorReading {
	return &sensorv1.SensorReading{
		SensorId:  r.SensorID,
		Room:      r.Room,
		Type:      toProtoSensorType(r.Type),
		Value:     r.Value,
		Unit:      r.Unit,
		Timestamp: timestamppb.New(r.Timestamp),
	}
}

func toProtoSensorType(t warden.SensorType) sensorv1.SensorType {
	switch t {
	case warden.Temperature:
		return sensorv1.SensorType_SENSOR_TYPE_TEMPERATURE
	case warden.Humidity:
		return sensorv1.SensorType_SENSOR_TYPE_HUMIDITY
	case warden.Motion:
		return sensorv1.SensorType_SENSOR_TYPE_MOTION
	case warden.CO2:
		return sensorv1.SensorType_SENSOR_TYPE_CO2
	default:
		return sensorv1.SensorType_SENSOR_TYPE_UNSPECIFIED
	}
}
