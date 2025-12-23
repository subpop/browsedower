package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/watchtower/web/middleware"
	"github.com/watchtower/web/models"
	"github.com/watchtower/web/services"
)

type PushSubscriptionRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

type NotificationPrefsRequest struct {
	NotifyNewRequests  bool `json:"notify_new_requests"`
	NotifyDeviceStatus bool `json:"notify_device_status"`
}

// GetVAPIDPublicKey returns the VAPID public key for push subscription
func GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	if services.Push == nil {
		http.Error(w, "Push notifications not configured", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"publicKey": services.Push.GetVAPIDPublicKey(),
	})
}

// SubscribePush registers a new push subscription for the current user
func SubscribePush(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req PushSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	sub, err := models.CreatePushSubscription(user.ID, req.Endpoint, req.Keys.P256dh, req.Keys.Auth)
	if err != nil {
		http.Error(w, "Failed to create subscription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sub)
}

// UnsubscribePush removes a push subscription
func UnsubscribePush(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := models.DeletePushSubscription(req.Endpoint); err != nil {
		http.Error(w, "Failed to delete subscription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetNotificationPrefs returns the current user's notification preferences
func GetNotificationPrefs(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"notify_new_requests":  user.NotifyNewRequests,
		"notify_device_status": user.NotifyDeviceStatus,
	})
}

// UpdateNotificationPrefs updates the current user's notification preferences
func UpdateNotificationPrefs(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req NotificationPrefsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := models.UpdateUserNotificationPrefs(user.ID, req.NotifyNewRequests, req.NotifyDeviceStatus); err != nil {
		http.Error(w, "Failed to update preferences", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetPushSubscriptions returns the current user's push subscriptions
func GetPushSubscriptions(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subs, err := models.GetPushSubscriptionsByUser(user.ID)
	if err != nil {
		http.Error(w, "Failed to get subscriptions", http.StatusInternalServerError)
		return
	}

	if subs == nil {
		subs = []models.PushSubscription{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subscriptions": subs,
	})
}

