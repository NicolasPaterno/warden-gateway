package hub

import (
	"context"
	"math"
	"os"
	"testing"
	"time"

	"github.com/nicaozx/warden-gateway"
	"github.com/nicaozx/warden-gateway/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMain(m *testing.M) {
	_ = metrics.Register()
	os.Exit(m.Run())
}

// createTestClient builds a Client with a nil conn — safe for tests that
// only exercise the hub's channel logic, never the WebSocket write path.
func createTestClient(h *Hub, bufSize int) *Client {
	return &Client{
		hub:  h,
		conn: nil,
		send: make(chan []byte, bufSize),
	}
}

// register one client, assert it appears in h.clients
func TestHub_RegisterClient(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	client := createTestClient(hub, 64)
	hub.register <- client

	reading := warden.SensorReading{Type: warden.Temperature, Value: 22.5}
	err := hub.Broadcast(reading)
	if err != nil {
		t.Error(err)
	}

	receiveWithTimeout(t, client.send, time.Second)
}

// assert it's removed from h.clients and send channel is closed
func TestHub_UnregisterClient(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	client := createTestClient(hub, 64)
	hub.register <- client

	reading := warden.SensorReading{Type: warden.Temperature, Value: 22.5}
	err := hub.Broadcast(reading)
	if err != nil {
		t.Error(err)
	}

	receiveWithTimeout(t, client.send, time.Second)

	hub.unregister <- client
	select {
	case _, ok := <-client.send:
		if ok {
			t.Error("client should be closed")
		}
	case <-time.After(time.Second):
		t.Error("client never processed the unregister")
	}
}

// assert both clients receive the message on their send channel
func TestHub_BroadcastDelivery(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	client1 := createTestClient(hub, 64)
	client2 := createTestClient(hub, 64)
	hub.register <- client1
	hub.register <- client2
	err := hub.Broadcast(warden.SensorReading{Type: warden.Motion, Value: 1})
	if err != nil {
		t.Error(err)
	}
	receiveWithTimeout(t, client1.send, time.Second)
	receiveWithTimeout(t, client2.send, time.Second)
}

/*
broadcast a message, assert the slow client is removed and
other clients still receive the message
*/
func TestHub_SlowClientDropped(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Unbuffered channel: hub's inner select hits `default` because the test goroutine
	// is not receiving on this channel while the hub processes the broadcast.
	slowClient := createTestClient(hub, 0)
	normalClient := createTestClient(hub, 64)
	hub.register <- slowClient
	hub.register <- normalClient

	err := hub.Broadcast(warden.SensorReading{Type: warden.Humidity, Value: 70})
	if err != nil {
		t.Error(err)
	}

	// Sync barrier: Broadcast() returns when the hub *receives* the message, not after
	// it finishes the client for-loop. Registering a dummy client blocks until the hub
	// completes that for-loop — only then can it pick up the next register event.
	// This guarantees slowClient.send is already closed before we read it below.
	hub.register <- createTestClient(hub, 1)

	select {
	case _, ok := <-slowClient.send:
		if ok {
			t.Error("slow client should have been dropped, got a message instead")
		}
	case <-time.After(time.Second):
		t.Error("slow client was never dropped")
	}

	receiveWithTimeout(t, normalClient.send, time.Second)
}

// assert Broadcast returns an error when json.Marshal fails
func TestHub_Broadcast_InvalidReading(t *testing.T) {
	hub := NewHub()

	// NaN has no JSON representation — json.Marshal returns an error before the
	// reading ever reaches the hub's broadcast channel.
	err := hub.Broadcast(warden.SensorReading{Value: math.NaN()})
	if err == nil {
		t.Error("expected error for NaN value, got nil")
	}
}

// TestHub_RegisterClient_IncrementsGauge verifies that the WSClientsConnected
// Prometheus gauge increases by 1 when a client registers with the hub.
func TestHub_RegisterClient_IncrementsGauge(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	before := testutil.ToFloat64(metrics.WSClientsConnected)

	client := createTestClient(hub, 64)
	hub.register <- client

	// Sync barrier: the hub processes events sequentially. A successful broadcast+receive
	// guarantees the preceding register was already handled — no sleep needed.
	_ = hub.Broadcast(warden.SensorReading{Type: warden.Temperature, Value: 22.5})
	receiveWithTimeout(t, client.send, time.Second)

	after := testutil.ToFloat64(metrics.WSClientsConnected)
	if after != before+1 {
		t.Errorf("WSClientsConnected: expected %.0f after register, got %.0f", before+1, after)
	}
}

// TestHub_UnregisterClient_DecrementsGauge verifies that the WSClientsConnected
// Prometheus gauge decreases by 1 when a client unregisters from the hub.
func TestHub_UnregisterClient_DecrementsGauge(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	client := createTestClient(hub, 64)
	hub.register <- client

	// Sync: ensure the register was processed before reading the gauge.
	_ = hub.Broadcast(warden.SensorReading{Type: warden.Temperature, Value: 22.5})
	receiveWithTimeout(t, client.send, time.Second)

	afterRegister := testutil.ToFloat64(metrics.WSClientsConnected)

	hub.unregister <- client

	// Sync: the hub closes client.send as part of unregister processing.
	// Receiving the channel close confirms the hub finished the unregister case.
	select {
	case _, ok := <-client.send:
		if ok {
			t.Fatal("expected send channel to be closed after unregister")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout: hub did not process unregister")
	}

	afterUnregister := testutil.ToFloat64(metrics.WSClientsConnected)
	if afterUnregister != afterRegister-1 {
		t.Errorf("WSClientsConnected: expected %.0f after unregister, got %.0f", afterRegister-1, afterUnregister)
	}
}

// helper: drains one message from a channel with timeout
func receiveWithTimeout(t *testing.T, ch <-chan []byte, timeout time.Duration) []byte {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(timeout):
		t.Fatal("timeout: no message received")
		return nil
	}
}
