package nats_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nicaozx/warden-gateway"
	"github.com/nicaozx/warden-gateway/internal/nats"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPublisher_Publish(t *testing.T) {
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
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate: %v", err)
		}
	}()

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "4222")
	if err != nil {
		t.Fatal(err)
	}

	url := fmt.Sprintf("nats://%s:%s", host, port.Port())
	publish, err := nats.NewPublisher(url)
	if err != nil {
		t.Fatal(err)
	}

	nc, err := natsgo.Connect(url)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	reading := warden.SensorReading{
		SensorID:  "sensor1",
		Room:      "bathroom2",
		Type:      warden.Motion,
		Timestamp: time.Now(),
	}

	// Subscribe
	subject := fmt.Sprintf("warden.sensors.v1.%s.%s", reading.Room, reading.Type)
	sub, err := nc.SubscribeSync(subject)
	if err != nil {
		log.Fatal(err)
	}

	//publish the message
	err = publish.Publish(ctx, reading)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for a message
	msg, err := sub.NextMsg(10 * time.Second)
	if err != nil {
		log.Fatal(err)
	}

	// Use the response
	var received warden.SensorReading
	err = json.Unmarshal(msg.Data, &received)
	if err != nil {
		t.Fatal(err)
	}
	if received.SensorID != reading.SensorID || received.Room != reading.Room || received.Type != reading.Type {
		t.Errorf("received wrong reading")
	}
}
