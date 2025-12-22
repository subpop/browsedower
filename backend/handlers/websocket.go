package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/browsedower/web/models"
	"github.com/browsedower/web/websocket"
	ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from browser extensions
		return true
	},
}

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// HandleWebSocket handles WebSocket connections from browser extensions
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	// Validate token and get device
	device, err := models.GetDeviceByToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create client
	client := &websocket.Client{
		Hub:         websocket.DefaultHub,
		Conn:        conn,
		DeviceID:    device.ID,
		DeviceToken: token,
		Send:        make(chan []byte, 256),
	}

	// Register client with hub
	websocket.DefaultHub.Register(client)

	// Update device heartbeat on connection
	if err := models.UpdateDeviceHeartbeat(device.ID); err != nil {
		log.Printf("Failed to update heartbeat on WS connect for device %d: %v", device.ID, err)
	}

	// Send initial patterns
	go sendInitialPatterns(client, device.ID)

	// Start read and write pumps
	go writePump(client)
	go readPump(client)
}

// sendInitialPatterns sends the current patterns to a newly connected client
func sendInitialPatterns(client *websocket.Client, deviceID int64) {
	patterns, err := models.GetPatternsByDevice(deviceID)
	if err != nil {
		log.Printf("Failed to get patterns for device %d: %v", deviceID, err)
		return
	}

	if patterns == nil {
		patterns = []models.Pattern{}
	}

	message := websocket.Message{
		Type: "patterns_updated",
		Data: map[string]interface{}{
			"patterns": patterns,
		},
	}

	websocket.DefaultHub.SendToDevice(deviceID, message)
}

// readPump pumps messages from the WebSocket connection to the hub
func readPump(client *websocket.Client) {
	defer func() {
		websocket.DefaultHub.Unregister(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(maxMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		// Update device heartbeat on each pong (device is still connected)
		if err := models.UpdateDeviceHeartbeat(client.DeviceID); err != nil {
			log.Printf("Failed to update heartbeat on pong for device %d: %v", client.DeviceID, err)
		}
		return nil
	})

	for {
		_, _, err := client.Conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		// We don't process incoming messages from clients for now
		// This could be extended to handle sync requests, etc.
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func writePump(client *websocket.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				client.Conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(ws.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// NotifyDevicePatternUpdate sends a pattern update notification to a device
func NotifyDevicePatternUpdate(deviceID int64) {
	if websocket.DefaultHub == nil {
		return
	}

	patterns, err := models.GetPatternsByDevice(deviceID)
	if err != nil {
		log.Printf("Failed to get patterns for device %d: %v", deviceID, err)
		return
	}

	if patterns == nil {
		patterns = []models.Pattern{}
	}

	message := websocket.Message{
		Type: "patterns_updated",
		Data: map[string]interface{}{
			"patterns": patterns,
		},
	}

	websocket.DefaultHub.SendToDevice(deviceID, message)
}

