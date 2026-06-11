package hub

import (
	"context"
	"encoding/json"

	"github.com/NicolasPaterno/warden-gateway"
	"github.com/NicolasPaterno/warden-gateway/internal/metrics"
	"github.com/gorilla/websocket"
)

type broadcastMessage struct {
	tenantID string
	data     []byte
}

type Hub struct {
	clients    map[string]map[*Client]bool
	broadcast  chan broadcastMessage
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan broadcastMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			if h.clients[client.tenantID] == nil {
				h.clients[client.tenantID] = make(map[*Client]bool)
			}
			h.clients[client.tenantID][client] = true
			metrics.WSClientsConnected.Inc()
		case client := <-h.unregister:
			if conns, ok := h.clients[client.tenantID]; ok {
				if _, ok := conns[client]; ok {
					delete(conns, client)
					close(client.send)
					metrics.WSClientsConnected.Dec()
					if len(conns) == 0 {
						delete(h.clients, client.tenantID)
					}
				}
			}
		case message := <-h.broadcast:
			for client := range h.clients[message.tenantID] {
				select {
				case client.send <- message.data:
				default:
					close(client.send)
					delete(h.clients[message.tenantID], client)
					metrics.WSClientsConnected.Dec()
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *Hub) Broadcast(reading warden.SensorReading) error {
	message, err := json.Marshal(reading)
	if err != nil {
		return err
	}
	h.broadcast <- broadcastMessage{tenantID: reading.TenantID, data: message}
	return nil
}

func (h *Hub) ServeWs(conn *websocket.Conn, tenantID string) {
	client := NewClient(h, conn, tenantID)
	h.register <- client
	go client.readPump()
	go client.writePump()
}
