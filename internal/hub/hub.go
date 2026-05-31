package hub

import (
	"context"
	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/nicaozx/warden-gateway"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
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
	h.broadcast <- message
	return nil
}

func (h *Hub) ServeWs(conn *websocket.Conn) {
	client := NewClient(h, conn)
	h.register <- client
	go client.readPump()
	go client.writePump()
}
