package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/browsedower/web/middleware"
	"github.com/browsedower/web/models"
	"github.com/browsedower/web/services"
	"github.com/gorilla/mux"
)

type AccessRequest struct {
	URL              string `json:"url"`
	SuggestedPattern string `json:"suggested_pattern,omitempty"`
}

type RequestsResponse struct {
	Requests []models.Request `json:"requests"`
}

type ApproveRequestBody struct {
	Pattern       string `json:"pattern"`
	Type          string `json:"type"`                     // "allow" or "deny"
	Duration      string `json:"duration"`                 // preset durations or "custom"
	CustomMinutes int    `json:"custom_minutes,omitempty"` // minutes for custom duration
}

// CreateRequest handles access requests from extensions
func CreateRequest(w http.ResponseWriter, r *http.Request) {
	device := middleware.GetDeviceFromContext(r)
	if device == nil {
		http.Error(w, "Device not found", http.StatusUnauthorized)
		return
	}

	var req AccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	accessReq, err := models.CreateRequest(device.ID, req.URL, req.SuggestedPattern)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Send push notification for new request
	if services.Push != nil {
		go services.Push.NotifyNewRequest(device.Name, req.URL)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(accessReq)
}

// ListRequests returns all access requests (admin API)
func ListRequests(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	requests, err := models.ListRequests(status)
	if err != nil {
		http.Error(w, "Failed to get requests", http.StatusInternalServerError)
		return
	}

	if requests == nil {
		requests = []models.Request{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RequestsResponse{Requests: requests})
}

// ApproveRequest approves an access request and creates a pattern
func ApproveRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	var body ApproveRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.Pattern == "" {
		http.Error(w, "Pattern is required", http.StatusBadRequest)
		return
	}

	if body.Type == "" {
		body.Type = "allow"
	}

	// Get the request to find device_id
	accessReq, err := models.GetRequestByID(id)
	if err != nil {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}

	// Calculate expiration (use UTC for consistent timezone handling)
	var expiresAt *time.Time
	if body.Duration != "" && body.Duration != "permanent" {
		duration, ok := parseDuration(body.Duration, body.CustomMinutes)
		if !ok {
			http.Error(w, "Invalid duration", http.StatusBadRequest)
			return
		}
		if duration > 0 {
			t := time.Now().UTC().Add(duration)
			expiresAt = &t
		}
	}

	// Create the pattern
	pattern, err := models.CreatePattern(accessReq.DeviceID, body.Pattern, body.Type, expiresAt)
	if err != nil {
		http.Error(w, "Failed to create pattern", http.StatusInternalServerError)
		return
	}

	// Mark request as approved
	if err := models.ApproveRequest(id); err != nil {
		http.Error(w, "Failed to update request", http.StatusInternalServerError)
		return
	}

	// Notify device via WebSocket
	go NotifyDevicePatternUpdate(accessReq.DeviceID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"pattern": pattern,
	})
}

// DenyRequest denies an access request
func DenyRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	if err := models.DenyRequest(id); err != nil {
		http.Error(w, "Failed to deny request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

