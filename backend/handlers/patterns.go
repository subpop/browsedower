package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/browsedower/web/middleware"
	"github.com/browsedower/web/models"
	"github.com/gorilla/mux"
)

type PatternResponse struct {
	Patterns []models.Pattern `json:"patterns"`
}

type CreatePatternRequest struct {
	DeviceID      int64  `json:"device_id"`
	Pattern       string `json:"pattern"`
	Type          string `json:"type"`
	Duration      string `json:"duration,omitempty"`       // preset durations or "custom"
	CustomMinutes int    `json:"custom_minutes,omitempty"` // minutes for custom duration
}

type UpdatePatternRequest struct {
	Pattern       string `json:"pattern"`
	Type          string `json:"type"`
	Duration      string `json:"duration,omitempty"`       // preset durations or "custom"
	CustomMinutes int    `json:"custom_minutes,omitempty"` // minutes for custom duration
}

// parseDuration converts duration string to time.Duration
// Supports: "15m", "30m", "1h", "8h", "24h", "1w", "permanent", "custom"
func parseDuration(duration string, customMinutes int) (time.Duration, bool) {
	switch duration {
	case "15m":
		return 15 * time.Minute, true
	case "30m":
		return 30 * time.Minute, true
	case "1h":
		return time.Hour, true
	case "8h":
		return 8 * time.Hour, true
	case "24h":
		return 24 * time.Hour, true
	case "1w":
		return 7 * 24 * time.Hour, true
	case "custom":
		if customMinutes > 0 {
			return time.Duration(customMinutes) * time.Minute, true
		}
		return 0, false
	case "permanent", "":
		return 0, true
	default:
		return 0, false
	}
}

// GetPatterns returns patterns for the authenticated device (extension API)
func GetPatterns(w http.ResponseWriter, r *http.Request) {
	device := middleware.GetDeviceFromContext(r)
	if device == nil {
		http.Error(w, "Device not found", http.StatusUnauthorized)
		return
	}

	patterns, err := models.GetPatternsByDevice(device.ID)
	if err != nil {
		http.Error(w, "Failed to get patterns", http.StatusInternalServerError)
		return
	}

	if patterns == nil {
		patterns = []models.Pattern{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PatternResponse{Patterns: patterns})
}

// ListAllPatterns returns all patterns (admin API)
func ListAllPatterns(w http.ResponseWriter, r *http.Request) {
	patterns, err := models.ListAllPatterns()
	if err != nil {
		http.Error(w, "Failed to get patterns", http.StatusInternalServerError)
		return
	}

	if patterns == nil {
		patterns = []models.Pattern{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PatternResponse{Patterns: patterns})
}

// CreatePattern creates a new pattern (admin API)
func CreatePattern(w http.ResponseWriter, r *http.Request) {
	var req CreatePatternRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Pattern == "" || req.Type == "" || req.DeviceID == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if req.Type != "allow" && req.Type != "deny" {
		http.Error(w, "Invalid pattern type", http.StatusBadRequest)
		return
	}

	var expiresAt *time.Time
	if req.Duration != "" && req.Duration != "permanent" {
		duration, ok := parseDuration(req.Duration, req.CustomMinutes)
		if !ok {
			http.Error(w, "Invalid duration", http.StatusBadRequest)
			return
		}
		if duration > 0 {
			t := time.Now().UTC().Add(duration)
			expiresAt = &t
		}
	}

	pattern, err := models.CreatePattern(req.DeviceID, req.Pattern, req.Type, expiresAt)
	if err != nil {
		http.Error(w, "Failed to create pattern", http.StatusInternalServerError)
		return
	}

	// Notify device via WebSocket
	go NotifyDevicePatternUpdate(req.DeviceID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pattern)
}

// UpdatePattern updates an existing pattern (admin API)
func UpdatePattern(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid pattern ID", http.StatusBadRequest)
		return
	}

	// Get existing pattern to find device ID
	existingPattern, err := models.GetPatternByID(id)
	if err != nil {
		http.Error(w, "Pattern not found", http.StatusNotFound)
		return
	}

	var req UpdatePatternRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Pattern == "" || req.Type == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if req.Type != "allow" && req.Type != "deny" {
		http.Error(w, "Invalid pattern type", http.StatusBadRequest)
		return
	}

	var expiresAt *time.Time
	if req.Duration != "" && req.Duration != "permanent" {
		duration, ok := parseDuration(req.Duration, req.CustomMinutes)
		if !ok {
			http.Error(w, "Invalid duration", http.StatusBadRequest)
			return
		}
		if duration > 0 {
			t := time.Now().UTC().Add(duration)
			expiresAt = &t
		}
	}

	pattern, err := models.UpdatePattern(id, req.Pattern, req.Type, expiresAt)
	if err != nil {
		http.Error(w, "Failed to update pattern", http.StatusInternalServerError)
		return
	}

	// Notify device via WebSocket
	go NotifyDevicePatternUpdate(existingPattern.DeviceID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pattern)
}

// DeletePattern removes a pattern (admin API)
func DeletePattern(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid pattern ID", http.StatusBadRequest)
		return
	}

	// Get pattern to find device ID before deletion
	pattern, err := models.GetPatternByID(id)
	if err != nil {
		http.Error(w, "Pattern not found", http.StatusNotFound)
		return
	}
	deviceID := pattern.DeviceID

	if err := models.DeletePattern(id); err != nil {
		http.Error(w, "Failed to delete pattern", http.StatusInternalServerError)
		return
	}

	// Notify device via WebSocket
	go NotifyDevicePatternUpdate(deviceID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// TogglePattern enables or disables a pattern (admin API)
func TogglePattern(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid pattern ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := models.TogglePatternEnabled(id, req.Enabled); err != nil {
		http.Error(w, "Failed to update pattern", http.StatusInternalServerError)
		return
	}

	pattern, err := models.GetPatternByID(id)
	if err != nil {
		http.Error(w, "Failed to get pattern", http.StatusInternalServerError)
		return
	}

	// Notify device via WebSocket
	go NotifyDevicePatternUpdate(pattern.DeviceID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pattern)
}

