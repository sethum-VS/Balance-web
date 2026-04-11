package websocket

import (
	"balance-web/internal/domain"
	"sync"
)

// Client represents a single connected active WebSocket endpoint.
type Client struct {
	ID         string
	DeviceType string
	Send       chan *domain.WSEvent // Passes event structs instead of raw byte slices
}

// Hub maintains the set of active WebSocket clients and processes broadcasts.
type Hub struct {
	Clients     map[string]*Client
	Register    chan *Client
	Unregister  chan *Client
	Broadcast   chan *domain.WSEvent
	mobileCount int
	mu          sync.RWMutex
}

// NewHub allocates a new Hub.
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *domain.WSEvent),
	}
}

// IsMobileOnline safely returns whether at least one iOS client is connected.
func (h *Hub) IsMobileOnline() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.mobileCount > 0
}

func (h *Hub) broadcastMobileStatus() {
	statusEvent := &domain.WSEvent{
		Type: domain.EventMobileStatus,
		Payload: map[string]interface{}{
			"isOnline": h.mobileCount > 0,
		},
	}
	
	// Send to all non-iOS clients (web)
	for _, client := range h.Clients {
		if client.DeviceType != "iOS" {
			select {
			case client.Send <- statusEvent:
			default:
				close(client.Send)
				delete(h.Clients, client.ID)
			}
		}
	}
}

// Run listens to the Hub's channels and safely processes additions, removals, and broadcasts.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client.ID] = client
			if client.DeviceType == "iOS" {
				h.mobileCount++
				if h.mobileCount == 1 {
					h.broadcastMobileStatus()
				}
			}
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.ID]; ok {
				delete(h.Clients, client.ID)
				close(client.Send)
				if client.DeviceType == "iOS" {
					h.mobileCount--
					if h.mobileCount == 0 {
						h.broadcastMobileStatus()
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.Broadcast:
			h.mu.RLock()
			for _, client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}
