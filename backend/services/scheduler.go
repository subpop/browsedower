package services

import (
	"log"
	"time"

	"github.com/watchtower/web/models"
)

// StartScheduler starts background tasks
func StartScheduler() {
	// Check for inactive devices every 1 minute for responsive status updates
	go func() {
		// Run immediately on startup
		checkInactiveDevices()
		
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			checkInactiveDevices()
		}
	}()

	log.Println("Background scheduler started")
}

// checkInactiveDevices marks devices inactive if no heartbeat for 2 minutes
// WebSocket ping/pong happens every ~54 seconds, so 2 minutes gives some buffer
func checkInactiveDevices() {
	deviceNames, err := models.MarkInactiveDevices(2 * time.Minute)
	if err != nil {
		log.Printf("Error marking inactive devices: %v", err)
		return
	}

	// Send notifications for newly inactive devices
	if Push != nil {
		for _, name := range deviceNames {
			go Push.NotifyDeviceStatus(name, "inactive")
		}
	}

	if len(deviceNames) > 0 {
		log.Printf("Marked %d devices as inactive", len(deviceNames))
	}
}

