package nats_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nicaozx/warden-gateway"
	"github.com/nicaozx/warden-gateway/internal/nats"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestMain(m *testing.M) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	os.Exit(m.Run())
}

// setupNATSContainer starts a real NATS server in Docker and returns the connection URL.
// The container is automatically terminated when the test ends via t.Cleanup.
func setupNATSContainer(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:latest",
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForLog("Server is ready"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate NATS container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "4222")
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("nats://%s:%s", host, port.Port())
}

// TestPublisher_Publish verifies that a SensorReading is published to the correct
// NATS subject and arrives with the expected payload.
func TestPublisher_Publish(t *testing.T) {
	url := setupNATSContainer(t)
	ctx := context.Background()

	publisher, err := nats.NewPublisher(url)
	if err != nil {
		t.Fatal(err)
	}

	nc, err := natsgo.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()

	reading := warden.SensorReading{
		SensorID:  "sensor1",
		Room:      "bathroom2",
		Type:      warden.Motion,
		Timestamp: time.Now(),
	}

	subject := fmt.Sprintf("warden.sensors.v1.%s.%s", reading.Room, reading.Type)
	sub, err := nc.SubscribeSync(subject)
	if err != nil {
		t.Fatal(err)
	}
	// Flush ensures the SUB command reaches the server before we publish.
	// Without this, PublishMsg can race ahead of the subscription registration.
	if err = nc.Flush(); err != nil {
		t.Fatal(err)
	}

	if err = publisher.Publish(ctx, reading); err != nil {
		t.Fatal(err)
	}

	msg, err := sub.NextMsg(10 * time.Second)
	if err != nil {
		t.Fatal(err)
	}

	var received warden.SensorReading
	if err = json.Unmarshal(msg.Data, &received); err != nil {
		t.Fatal(err)
	}
	if received.SensorID != reading.SensorID || received.Room != reading.Room || received.Type != reading.Type {
		t.Errorf("received wrong reading: got %+v, want %+v", received, reading)
	}
}

// TestPublisher_Publish_PropagatesTraceContext verifies that when Publish is called
// with a context that has an active span, the W3C traceparent header is injected
// into the NATS message and received by the subscriber.
//
// This is the contract the warden-engine relies on: it extracts the traceparent
// header from incoming NATS messages to continue the distributed trace.
func TestPublisher_Publish_PropagatesTraceContext(t *testing.T) {
	url := setupNATSContainer(t)

	publisher, err := nats.NewPublisher(url)
	if err != nil {
		t.Fatal(err)
	}

	nc, err := natsgo.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()

	reading := warden.SensorReading{
		SensorID:  "sensor-trace",
		Room:      "kitchen",
		Type:      warden.Temperature,
		Timestamp: time.Now(),
	}

	subject := fmt.Sprintf("warden.sensors.v1.%s.%s", reading.Room, reading.Type)
	sub, err := nc.SubscribeSync(subject)
	if err != nil {
		t.Fatal(err)
	}
	if err = nc.Flush(); err != nil {
		t.Fatal(err)
	}

	// Start a real span — the propagator will extract its trace ID and inject it
	// as "traceparent: 00-<traceID>-<spanID>-01" into the NATS message header.
	ctx, span := otel.Tracer("test").Start(context.Background(), "test-span")
	defer span.End()
	traceID := span.SpanContext().TraceID().String()

	if err = publisher.Publish(ctx, reading); err != nil {
		t.Fatal(err)
	}

	msg, err := sub.NextMsg(10 * time.Second)
	if err != nil {
		t.Fatal(err)
	}

	traceparent := msg.Header.Get("traceparent")
	if traceparent == "" {
		t.Fatal("expected traceparent header in NATS message, got empty string")
	}
	if !strings.Contains(traceparent, traceID) {
		t.Errorf("traceparent %q does not contain expected trace ID %q", traceparent, traceID)
	}
}
