package services

import (
	"encoding/json"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/watchtower/web/models"
)

// PushService handles sending web push notifications
type PushService struct {
	vapidPublicKey  string
	vapidPrivateKey string
}

// NotificationPayload represents the data sent in a push notification
type NotificationPayload struct {
	Title   string `json:"title"`
	Body    string `json:"body"`
	Icon    string `json:"icon,omitempty"`
	URL     string `json:"url,omitempty"`
	Tag     string `json:"tag,omitempty"`
	Type    string `json:"type"` // "new_request" or "device_status"
}

var Push *PushService

// InitPushService initializes the push notification service
func InitPushService() error {
	// Try to load existing VAPID keys
	publicKey, err := models.GetAppConfig("vapid_public_key")
	if err != nil {
		// Generate new keys
		privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			return err
		}

		if err := models.SetAppConfig("vapid_public_key", publicKey); err != nil {
			return err
		}
		if err := models.SetAppConfig("vapid_private_key", privateKey); err != nil {
			return err
		}

		Push = &PushService{
			vapidPublicKey:  publicKey,
			vapidPrivateKey: privateKey,
		}
		log.Println("Generated new VAPID keys for push notifications")
		return nil
	}

	privateKey, err := models.GetAppConfig("vapid_private_key")
	if err != nil {
		return err
	}

	Push = &PushService{
		vapidPublicKey:  publicKey,
		vapidPrivateKey: privateKey,
	}
	log.Println("Push notification service initialized")
	return nil
}

// GetVAPIDPublicKey returns the public VAPID key for client subscription
func (p *PushService) GetVAPIDPublicKey() string {
	return p.vapidPublicKey
}

// SendNotification sends a push notification to a specific subscription
func (p *PushService) SendNotification(sub *models.PushSubscription, payload NotificationPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	subscription := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	resp, err := webpush.SendNotification(payloadBytes, subscription, &webpush.Options{
		Subscriber:      "admin@watchtower.local",
		VAPIDPublicKey:  p.vapidPublicKey,
		VAPIDPrivateKey: p.vapidPrivateKey,
		TTL:             60,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If subscription is expired or invalid, remove it
	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		log.Printf("Removing invalid push subscription: %s", sub.Endpoint)
		models.DeletePushSubscription(sub.Endpoint)
	}

	return nil
}

// NotifyNewRequest sends notifications for a new access request
func (p *PushService) NotifyNewRequest(deviceName, url string) {
	users, err := models.GetUsersForNotification("new_request")
	if err != nil {
		log.Printf("Error getting users for notification: %v", err)
		return
	}

	payload := NotificationPayload{
		Title: "New Access Request",
		Body:  deviceName + " is requesting access to " + truncateURL(url),
		Icon:  "/admin/icon-192.png",
		URL:   "/admin/#requests",
		Tag:   "new-request",
		Type:  "new_request",
	}

	p.sendToUsers(users, payload)
}

// NotifyDeviceStatus sends notifications for device status changes
func (p *PushService) NotifyDeviceStatus(deviceName, status string) {
	users, err := models.GetUsersForNotification("device_status")
	if err != nil {
		log.Printf("Error getting users for notification: %v", err)
		return
	}

	var body string
	switch status {
	case "inactive":
		body = deviceName + " has gone inactive (no heartbeat)"
	case "uninstalled":
		body = deviceName + " extension has been uninstalled"
	default:
		body = deviceName + " status changed to " + status
	}

	payload := NotificationPayload{
		Title: "Device Status Change",
		Body:  body,
		Icon:  "/admin/icon-192.png",
		URL:   "/admin/#devices",
		Tag:   "device-status-" + deviceName,
		Type:  "device_status",
	}

	p.sendToUsers(users, payload)
}

func (p *PushService) sendToUsers(users []models.User, payload NotificationPayload) {
	for _, user := range users {
		subs, err := models.GetPushSubscriptionsByUser(user.ID)
		if err != nil {
			log.Printf("Error getting subscriptions for user %d: %v", user.ID, err)
			continue
		}

		for _, sub := range subs {
			go func(s models.PushSubscription) {
				if err := p.SendNotification(&s, payload); err != nil {
					log.Printf("Error sending push notification: %v", err)
				}
			}(sub)
		}
	}
}

func truncateURL(url string) string {
	if len(url) > 50 {
		return url[:47] + "..."
	}
	return url
}

