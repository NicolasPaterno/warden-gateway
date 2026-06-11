package nats

import (
	"context"
	"log/slog"
	"time"

	warden "github.com/NicolasPaterno/warden-gateway"
	sensorv1 "github.com/NicolasPaterno/warden-proto/gen/go/warden/sensor/v1"
	natsgo "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type Subscriber struct {
	conn *natsgo.Conn
	sub  *natsgo.Subscription
}

func NewSubscriber(url string) (*Subscriber, error) {
	conn, err := natsgo.Connect(
		url,
		natsgo.ReconnectHandler(func(conn *natsgo.Conn) {
			slog.Info("subscriber reconnected to NATS", "url", conn.ConnectedUrl())
		}),
		natsgo.MaxReconnects(10),
		natsgo.ReconnectWait(5*time.Second),
		natsgo.RetryOnFailedConnect(true),
		natsgo.DisconnectErrHandler(func(conn *natsgo.Conn, err error) {
			slog.Warn("subscriber disconnected from NATS", "error", err)
		}))
	if err != nil {
		return nil, err
	}
	return &Subscriber{conn: conn}, nil
}

func (s *Subscriber) Subscribe(ctx context.Context, subject string, handler func(warden.SensorReading)) error {
	sub, err := s.conn.Subscribe(subject, func(msg *natsgo.Msg) {
		var protoReading sensorv1.SensorReading
		if err := proto.Unmarshal(msg.Data, &protoReading); err != nil {
			slog.Error("failed to unmarshal sensor reading", "error", err)
			return
		}

		handler(fromProtoReading(&protoReading))
	})
	if err != nil {
		return err
	}
	s.sub = sub

	go func() {
		<-ctx.Done()
		if err := sub.Unsubscribe(); err != nil {
			slog.Warn("failed to unsubscribe", "error", err)
		}
	}()

	return nil
}

func (s *Subscriber) Close() error {
	return s.conn.Drain()
}

func fromProtoReading(r *sensorv1.SensorReading) warden.SensorReading {
	return warden.SensorReading{
		TenantID:  r.TenantId,
		SensorID:  r.SensorId,
		Room:      r.Room,
		Type:      fromProtoSensorType(r.Type),
		Value:     r.Value,
		Unit:      r.Unit,
		Timestamp: r.Timestamp.AsTime(),
	}
}

func fromProtoSensorType(t sensorv1.SensorType) warden.SensorType {
	switch t {
	case sensorv1.SensorType_SENSOR_TYPE_TEMPERATURE:
		return warden.Temperature
	case sensorv1.SensorType_SENSOR_TYPE_HUMIDITY:
		return warden.Humidity
	case sensorv1.SensorType_SENSOR_TYPE_MOTION:
		return warden.Motion
	case sensorv1.SensorType_SENSOR_TYPE_CO2:
		return warden.CO2
	default:
		return ""
	}
}
