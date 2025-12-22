package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/browsedower/web/database"
	"golang.org/x/crypto/bcrypt"
)

// User represents an admin user
type User struct {
	ID                 int64     `json:"id"`
	Username           string    `json:"username"`
	PasswordHash       string    `json:"-"`
	NotifyNewRequests  bool      `json:"notify_new_requests"`
	NotifyDeviceStatus bool      `json:"notify_device_status"`
	CreatedAt          time.Time `json:"created_at"`
}

// PushSubscription represents a web push subscription for a user
type PushSubscription struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	CreatedAt time.Time `json:"created_at"`
}

// Device represents a registered browser extension
type Device struct {
	ID        int64      `json:"id"`
	Token     string     `json:"token,omitempty"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`    // "active", "inactive", "uninstalled"
	LastSeen  *time.Time `json:"last_seen"` // Last heartbeat time
	CreatedAt time.Time  `json:"created_at"`
}

// Pattern represents an allow/deny URL pattern
type Pattern struct {
	ID        int64      `json:"id"`
	DeviceID  int64      `json:"device_id"`
	Pattern   string     `json:"pattern"`
	Type      string     `json:"type"` // "allow" or "deny"
	Enabled   bool       `json:"enabled"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Request represents an access request from an extension
type Request struct {
	ID               int64      `json:"id"`
	DeviceID         int64      `json:"device_id"`
	DeviceName       string     `json:"device_name,omitempty"`
	URL              string     `json:"url"`
	SuggestedPattern string     `json:"suggested_pattern,omitempty"`
	Status           string     `json:"status"` // "pending", "approved", "denied"
	CreatedAt        time.Time  `json:"created_at"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
}

// Session represents an admin session
type Session struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Helper function to generate random tokens
func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ========== User Operations ==========

func CreateUser(username, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	result, err := database.DB.Exec(
		"INSERT INTO users (username, password_hash, notify_new_requests, notify_device_status) VALUES (?, ?, 1, 1)",
		username, string(hash),
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &User{
		ID:                 id,
		Username:           username,
		NotifyNewRequests:  true,
		NotifyDeviceStatus: true,
		CreatedAt:          time.Now(),
	}, nil
}

func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := database.DB.QueryRow(
		"SELECT id, username, password_hash, COALESCE(notify_new_requests, 1), COALESCE(notify_device_status, 1), created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.NotifyNewRequests, &user.NotifyDeviceStatus, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := database.DB.QueryRow(
		"SELECT id, username, password_hash, COALESCE(notify_new_requests, 1), COALESCE(notify_device_status, 1), created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.NotifyNewRequests, &user.NotifyDeviceStatus, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func ListUsers() ([]User, error) {
	rows, err := database.DB.Query("SELECT id, username, COALESCE(notify_new_requests, 1), COALESCE(notify_device_status, 1), created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.NotifyNewRequests, &u.NotifyDeviceStatus, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func UpdateUserNotificationPrefs(userID int64, notifyNewRequests, notifyDeviceStatus bool) error {
	_, err := database.DB.Exec(
		"UPDATE users SET notify_new_requests = ?, notify_device_status = ? WHERE id = ?",
		notifyNewRequests, notifyDeviceStatus, userID,
	)
	return err
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

func UpdateUserPassword(userID int64, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = database.DB.Exec(
		"UPDATE users SET password_hash = ? WHERE id = ?",
		string(hash), userID,
	)
	return err
}

// ========== Device Operations ==========

func CreateDevice(name string) (*Device, error) {
	token, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	result, err := database.DB.Exec(
		"INSERT INTO devices (token, name, status, last_seen) VALUES (?, ?, 'active', ?)",
		token, name, now,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Device{
		ID:        id,
		Token:     token,
		Name:      name,
		Status:    "active",
		LastSeen:  &now,
		CreatedAt: now,
	}, nil
}

func GetDeviceByToken(token string) (*Device, error) {
	device := &Device{}
	err := database.DB.QueryRow(
		"SELECT id, token, name, COALESCE(status, 'active'), last_seen, created_at FROM devices WHERE token = ?",
		token,
	).Scan(&device.ID, &device.Token, &device.Name, &device.Status, &device.LastSeen, &device.CreatedAt)
	if err != nil {
		return nil, err
	}
	return device, nil
}

func GetDeviceByID(id int64) (*Device, error) {
	device := &Device{}
	err := database.DB.QueryRow(
		"SELECT id, token, name, COALESCE(status, 'active'), last_seen, created_at FROM devices WHERE id = ?",
		id,
	).Scan(&device.ID, &device.Token, &device.Name, &device.Status, &device.LastSeen, &device.CreatedAt)
	if err != nil {
		return nil, err
	}
	return device, nil
}

func RegenerateDeviceToken(id int64) (*Device, error) {
	token, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	_, err = database.DB.Exec("UPDATE devices SET token = ? WHERE id = ?", token, id)
	if err != nil {
		return nil, err
	}

	return GetDeviceByID(id)
}

func ListDevices() ([]Device, error) {
	rows, err := database.DB.Query("SELECT id, name, COALESCE(status, 'active'), last_seen, created_at FROM devices ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Name, &d.Status, &d.LastSeen, &d.CreatedAt); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func DeleteDevice(id int64) error {
	_, err := database.DB.Exec("DELETE FROM devices WHERE id = ?", id)
	return err
}

// UpdateDeviceHeartbeat updates the last_seen timestamp and sets status to active
func UpdateDeviceHeartbeat(deviceID int64) error {
	_, err := database.DB.Exec(
		"UPDATE devices SET last_seen = ?, status = 'active' WHERE id = ?",
		time.Now(), deviceID,
	)
	return err
}

// UpdateDeviceStatus updates the device status (active, inactive, uninstalled)
func UpdateDeviceStatus(deviceID int64, status string) error {
	_, err := database.DB.Exec(
		"UPDATE devices SET status = ? WHERE id = ?",
		status, deviceID,
	)
	return err
}

// MarkInactiveDevices marks devices as inactive if they haven't been seen recently
// Returns the names of devices that were marked inactive
func MarkInactiveDevices(threshold time.Duration) ([]string, error) {
	cutoff := time.Now().Add(-threshold)

	// First get the names of devices that will be marked inactive
	rows, err := database.DB.Query(
		"SELECT name FROM devices WHERE status = 'active' AND (last_seen IS NULL OR last_seen < ?)",
		cutoff,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deviceNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		deviceNames = append(deviceNames, name)
	}

	// Now update them
	_, err = database.DB.Exec(
		"UPDATE devices SET status = 'inactive' WHERE status = 'active' AND (last_seen IS NULL OR last_seen < ?)",
		cutoff,
	)
	if err != nil {
		return nil, err
	}

	return deviceNames, nil
}

// ========== Pattern Operations ==========

func CreatePattern(deviceID int64, pattern, patternType string, expiresAt *time.Time) (*Pattern, error) {
	var result sql.Result
	var err error

	if expiresAt != nil {
		result, err = database.DB.Exec(
			"INSERT INTO patterns (device_id, pattern, type, enabled, expires_at) VALUES (?, ?, ?, 1, ?)",
			deviceID, pattern, patternType, expiresAt,
		)
	} else {
		result, err = database.DB.Exec(
			"INSERT INTO patterns (device_id, pattern, type, enabled) VALUES (?, ?, ?, 1)",
			deviceID, pattern, patternType,
		)
	}

	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Pattern{
		ID:        id,
		DeviceID:  deviceID,
		Pattern:   pattern,
		Type:      patternType,
		Enabled:   true,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

func GetPatternsByDevice(deviceID int64) ([]Pattern, error) {
	// Compare using datetime() which normalizes the format for comparison
	// Only return enabled patterns that haven't expired
	rows, err := database.DB.Query(`
		SELECT id, device_id, pattern, type, COALESCE(enabled, 1), expires_at, created_at 
		FROM patterns 
		WHERE device_id = ? AND COALESCE(enabled, 1) = 1 AND (expires_at IS NULL OR datetime(expires_at) > datetime('now'))
		ORDER BY created_at DESC
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		var p Pattern
		if err := rows.Scan(&p.ID, &p.DeviceID, &p.Pattern, &p.Type, &p.Enabled, &p.ExpiresAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		patterns = append(patterns, p)
	}
	return patterns, nil
}

func ListAllPatterns() ([]Pattern, error) {
	rows, err := database.DB.Query(`
		SELECT id, device_id, pattern, type, COALESCE(enabled, 1), expires_at, created_at 
		FROM patterns 
		ORDER BY CASE type WHEN 'deny' THEN 0 ELSE 1 END, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		var p Pattern
		if err := rows.Scan(&p.ID, &p.DeviceID, &p.Pattern, &p.Type, &p.Enabled, &p.ExpiresAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		patterns = append(patterns, p)
	}
	return patterns, nil
}

func DeletePattern(id int64) error {
	_, err := database.DB.Exec("DELETE FROM patterns WHERE id = ?", id)
	return err
}

func TogglePatternEnabled(id int64, enabled bool) error {
	_, err := database.DB.Exec("UPDATE patterns SET enabled = ? WHERE id = ?", enabled, id)
	return err
}

func GetPatternByID(id int64) (*Pattern, error) {
	pattern := &Pattern{}
	err := database.DB.QueryRow(`
		SELECT id, device_id, pattern, type, COALESCE(enabled, 1), expires_at, created_at 
		FROM patterns WHERE id = ?
	`, id).Scan(&pattern.ID, &pattern.DeviceID, &pattern.Pattern, &pattern.Type, &pattern.Enabled, &pattern.ExpiresAt, &pattern.CreatedAt)
	if err != nil {
		return nil, err
	}
	return pattern, nil
}

func UpdatePattern(id int64, pattern, patternType string, expiresAt *time.Time) (*Pattern, error) {
	var err error

	if expiresAt != nil {
		_, err = database.DB.Exec(
			"UPDATE patterns SET pattern = ?, type = ?, expires_at = ? WHERE id = ?",
			pattern, patternType, expiresAt, id,
		)
	} else {
		_, err = database.DB.Exec(
			"UPDATE patterns SET pattern = ?, type = ?, expires_at = NULL WHERE id = ?",
			pattern, patternType, id,
		)
	}

	if err != nil {
		return nil, err
	}

	return GetPatternByID(id)
}

// ========== Request Operations ==========

func CreateRequest(deviceID int64, url, suggestedPattern string) (*Request, error) {
	result, err := database.DB.Exec(
		"INSERT INTO requests (device_id, url, suggested_pattern) VALUES (?, ?, ?)",
		deviceID, url, suggestedPattern,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Request{
		ID:               id,
		DeviceID:         deviceID,
		URL:              url,
		SuggestedPattern: suggestedPattern,
		Status:           "pending",
		CreatedAt:        time.Now(),
	}, nil
}

func GetRequestByID(id int64) (*Request, error) {
	req := &Request{}
	err := database.DB.QueryRow(`
		SELECT r.id, r.device_id, d.name, r.url, r.suggested_pattern, r.status, r.created_at, r.resolved_at
		FROM requests r
		JOIN devices d ON r.device_id = d.id
		WHERE r.id = ?
	`, id).Scan(&req.ID, &req.DeviceID, &req.DeviceName, &req.URL, &req.SuggestedPattern, &req.Status, &req.CreatedAt, &req.ResolvedAt)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func ListRequests(status string) ([]Request, error) {
	var rows *sql.Rows
	var err error

	if status != "" {
		rows, err = database.DB.Query(`
			SELECT r.id, r.device_id, d.name, r.url, r.suggested_pattern, r.status, r.created_at, r.resolved_at
			FROM requests r
			JOIN devices d ON r.device_id = d.id
			WHERE r.status = ?
			ORDER BY r.created_at DESC
		`, status)
	} else {
		rows, err = database.DB.Query(`
			SELECT r.id, r.device_id, d.name, r.url, r.suggested_pattern, r.status, r.created_at, r.resolved_at
			FROM requests r
			JOIN devices d ON r.device_id = d.id
			ORDER BY r.created_at DESC
		`)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []Request
	for rows.Next() {
		var r Request
		if err := rows.Scan(&r.ID, &r.DeviceID, &r.DeviceName, &r.URL, &r.SuggestedPattern, &r.Status, &r.CreatedAt, &r.ResolvedAt); err != nil {
			return nil, err
		}
		requests = append(requests, r)
	}
	return requests, nil
}

func ApproveRequest(id int64) error {
	now := time.Now()
	_, err := database.DB.Exec(
		"UPDATE requests SET status = 'approved', resolved_at = ? WHERE id = ?",
		now, id,
	)
	return err
}

func DenyRequest(id int64) error {
	now := time.Now()
	_, err := database.DB.Exec(
		"UPDATE requests SET status = 'denied', resolved_at = ? WHERE id = ?",
		now, id,
	)
	return err
}

// ========== Session Operations ==========

func CreateSession(userID int64) (*Session, error) {
	token, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	result, err := database.DB.Exec(
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID, token, expiresAt,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Session{
		ID:        id,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

func GetSessionByToken(token string) (*Session, error) {
	session := &Session{}
	err := database.DB.QueryRow(
		"SELECT id, user_id, token, expires_at, created_at FROM sessions WHERE token = ? AND expires_at > datetime('now')",
		token,
	).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func DeleteSession(token string) error {
	_, err := database.DB.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func CleanExpiredSessions() error {
	_, err := database.DB.Exec("DELETE FROM sessions WHERE expires_at <= datetime('now')")
	return err
}

// GetUserCount returns the number of users in the database
func GetUserCount() (int, error) {
	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// NeedsSetup returns true if no users exist (first-time setup needed)
func NeedsSetup() (bool, error) {
	count, err := GetUserCount()
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// ========== Push Subscription Operations ==========

func CreatePushSubscription(userID int64, endpoint, p256dh, auth string) (*PushSubscription, error) {
	// Upsert - replace existing subscription for this endpoint
	_, err := database.DB.Exec(
		"DELETE FROM push_subscriptions WHERE endpoint = ?",
		endpoint,
	)
	if err != nil {
		return nil, err
	}

	result, err := database.DB.Exec(
		"INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth) VALUES (?, ?, ?, ?)",
		userID, endpoint, p256dh, auth,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &PushSubscription{
		ID:        id,
		UserID:    userID,
		Endpoint:  endpoint,
		P256dh:    p256dh,
		Auth:      auth,
		CreatedAt: time.Now(),
	}, nil
}

func GetPushSubscriptionsByUser(userID int64) ([]PushSubscription, error) {
	rows, err := database.DB.Query(
		"SELECT id, user_id, endpoint, p256dh, auth, created_at FROM push_subscriptions WHERE user_id = ?",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func GetAllPushSubscriptions() ([]PushSubscription, error) {
	rows, err := database.DB.Query(
		"SELECT id, user_id, endpoint, p256dh, auth, created_at FROM push_subscriptions",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func DeletePushSubscription(endpoint string) error {
	_, err := database.DB.Exec("DELETE FROM push_subscriptions WHERE endpoint = ?", endpoint)
	return err
}

func DeletePushSubscriptionByID(id int64) error {
	_, err := database.DB.Exec("DELETE FROM push_subscriptions WHERE id = ?", id)
	return err
}

// GetUsersForNotification returns users who should receive a specific notification type
func GetUsersForNotification(notificationType string) ([]User, error) {
	var query string
	switch notificationType {
	case "new_request":
		query = "SELECT id, username, COALESCE(notify_new_requests, 1), COALESCE(notify_device_status, 1), created_at FROM users WHERE COALESCE(notify_new_requests, 1) = 1"
	case "device_status":
		query = "SELECT id, username, COALESCE(notify_new_requests, 1), COALESCE(notify_device_status, 1), created_at FROM users WHERE COALESCE(notify_device_status, 1) = 1"
	default:
		return nil, nil
	}

	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.NotifyNewRequests, &u.NotifyDeviceStatus, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// ========== App Config Operations (for VAPID keys) ==========

func GetAppConfig(key string) (string, error) {
	var value string
	err := database.DB.QueryRow("SELECT value FROM app_config WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func SetAppConfig(key, value string) error {
	_, err := database.DB.Exec(
		"INSERT OR REPLACE INTO app_config (key, value) VALUES (?, ?)",
		key, value,
	)
	return err
}
