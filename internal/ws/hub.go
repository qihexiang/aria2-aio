package ws

import (
	"log"
	"sync"
)

type Hub struct {
	mu        sync.RWMutex
	clients   map[*Client]struct{}
	broadcast chan Message
	register  chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = struct{}{}
			h.mu.Unlock()
			log.Printf("Hub: client registered, total=%d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("Hub: client unregistered, total=%d", len(h.clients))

		case message := <-h.broadcast:
			data := message.ToJSON()
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- data:
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, client)
					close(client.Send)
					h.mu.Unlock()
					h.mu.RLock()
					log.Printf("Hub: dropped slow client")
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(msg Message) {
	select {
	case h.broadcast <- msg:
	default:
		log.Printf("Hub: broadcast channel full, dropping message type=%s", msg.Type)
	}
}

func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}