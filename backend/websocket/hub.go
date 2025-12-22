package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client
type Client struct {
	Hub        *Hub
	Conn       *websocket.Conn
	DeviceID   int64
	DeviceToken string
	Send       chan []byte
}

// Hub manages all WebSocket connections
type Hub struct {
	// Clients by device ID for targeted messaging
	clients map[int64]map[*Client]bool
	
	// Register requests from clients
	register chan *Client
	
	// Unregister requests from clients
	unregister chan *Client
	
	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// Message represents a WebSocket message
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// Global hub instance
var DefaultHub *Hub

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int64]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.DeviceID] == nil {
				h.clients[client.DeviceID] = make(map[*Client]bool)
			}
			h.clients[client.DeviceID][client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client registered for device %d", client.DeviceID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.DeviceID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.clients, client.DeviceID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket client unregistered for device %d", client.DeviceID)
		}
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// SendToDevice sends a message to all clients for a specific device
func (h *Hub) SendToDevice(deviceID int64, message Message) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	h.mu.RLock()
	clients, ok := h.clients[deviceID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	h.mu.RLock()
	for client := range clients {
		select {
		case client.Send <- data:
		default:
			// Client buffer full, skip
			log.Printf("WebSocket client buffer full for device %d", deviceID)
		}
	}
	h.mu.RUnlock()
}

// GetConnectedDeviceCount returns the number of devices with active connections
func (h *Hub) GetConnectedDeviceCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// IsDeviceConnected checks if a device has any active connections
func (h *Hub) IsDeviceConnected(deviceID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[deviceID]
	return ok
}

// InitHub initializes the global hub and starts it
func InitHub() {
	DefaultHub = NewHub()
	go DefaultHub.Run()
	log.Println("WebSocket hub initialized")
}

