package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/browsedower/web/middleware"
	"github.com/browsedower/web/models"
	"github.com/browsedower/web/services"
	"github.com/gorilla/mux"
)

type CreateDeviceRequest struct {
	Name string `json:"name"`
}

type DevicesResponse struct {
	Devices []models.Device `json:"devices"`
}

type HeartbeatResponse struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
}

// ListDevices returns all registered devices (admin API)
func ListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := models.ListDevices()
	if err != nil {
		http.Error(w, "Failed to get devices", http.StatusInternalServerError)
		return
	}

	if devices == nil {
		devices = []models.Device{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DevicesResponse{Devices: devices})
}

// CreateDevice registers a new device and returns its token (admin API)
func CreateDevice(w http.ResponseWriter, r *http.Request) {
	var req CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Device name is required", http.StatusBadRequest)
		return
	}

	device, err := models.CreateDevice(req.Name)
	if err != nil {
		http.Error(w, "Failed to create device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(device)
}

// DeleteDevice removes a device and all its patterns (admin API)
func DeleteDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	if err := models.DeleteDevice(id); err != nil {
		http.Error(w, "Failed to delete device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// RegenerateDeviceToken generates a new token for a device (admin API)
func RegenerateDeviceToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	device, err := models.RegenerateDeviceToken(id)
	if err != nil {
		http.Error(w, "Failed to regenerate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

// DeviceHeartbeat receives heartbeat pings from extensions (extension API)
func DeviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	device := middleware.GetDeviceFromContext(r)
	if device == nil {
		http.Error(w, "Device not found", http.StatusUnauthorized)
		return
	}

	if err := models.UpdateDeviceHeartbeat(device.ID); err != nil {
		log.Printf("Failed to update heartbeat for device %d: %v", device.ID, err)
		http.Error(w, "Failed to update heartbeat", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HeartbeatResponse{
		Success: true,
		Status:  "active",
	})
}

// DeviceUninstall marks a device as uninstalled (extension API - called via uninstall URL)
func DeviceUninstall(w http.ResponseWriter, r *http.Request) {
	// This endpoint can be called via GET (from setUninstallURL) or POST
	// Token can come from query param for GET requests
	token := r.URL.Query().Get("token")
	
	if token == "" {
		// Try to get from Authorization header (POST requests)
		device := middleware.GetDeviceFromContext(r)
		if device != nil {
			if err := models.UpdateDeviceStatus(device.ID, "uninstalled"); err != nil {
				log.Printf("Failed to mark device %d as uninstalled: %v", device.ID, err)
			} else {
				log.Printf("Device %d (%s) marked as uninstalled", device.ID, device.Name)
				// Send push notification
				if services.Push != nil {
					go services.Push.NotifyDeviceStatus(device.Name, "uninstalled")
				}
			}
		}
	} else {
		// Get device by token from query param
		device, err := models.GetDeviceByToken(token)
		if err == nil && device != nil {
			if err := models.UpdateDeviceStatus(device.ID, "uninstalled"); err != nil {
				log.Printf("Failed to mark device %d as uninstalled: %v", device.ID, err)
			} else {
				log.Printf("Device %d (%s) marked as uninstalled", device.ID, device.Name)
				// Send push notification
				if services.Push != nil {
					go services.Push.NotifyDeviceStatus(device.Name, "uninstalled")
				}
			}
		}
	}

	// Always return success (don't leak information about valid tokens)
	// For GET requests, we can show a simple page
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Browsedower</title></head>
<body style="font-family: system-ui; padding: 2rem; text-align: center;">
<h1>Extension Uninstalled</h1>
<p>The Browsedower extension has been uninstalled from this device.</p>
</body>
</html>`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

