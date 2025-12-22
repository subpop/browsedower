package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/browsedower/web/models"
)

type contextKey string

const (
	DeviceContextKey  contextKey = "device"
	SessionContextKey contextKey = "session"
	UserContextKey    contextKey = "user"
)

// CORS middleware to handle cross-origin requests
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// TokenAuth validates device token from Authorization header
func TokenAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]
		device, err := models.GetDeviceByToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), DeviceContextKey, device)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// SessionAuth validates session from cookie
func SessionAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		session, err := models.GetSessionByToken(cookie.Value)
		if err != nil {
			http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
			return
		}

		user, err := models.GetUserByID(session.UserID)
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), SessionContextKey, session)
		ctx = context.WithValue(ctx, UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// SessionAuthMiddleware is a middleware version for subrouters
func SessionAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		session, err := models.GetSessionByToken(cookie.Value)
		if err != nil {
			http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
			return
		}

		user, err := models.GetUserByID(session.UserID)
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), SessionContextKey, session)
		ctx = context.WithValue(ctx, UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetDeviceFromContext retrieves the device from request context
func GetDeviceFromContext(r *http.Request) *models.Device {
	device, ok := r.Context().Value(DeviceContextKey).(*models.Device)
	if !ok {
		return nil
	}
	return device
}

// GetUserFromContext retrieves the user from request context
func GetUserFromContext(r *http.Request) *models.User {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// GetSessionFromContext retrieves the session from request context
func GetSessionFromContext(r *http.Request) *models.Session {
	session, ok := r.Context().Value(SessionContextKey).(*models.Session)
	if !ok {
		return nil
	}
	return session
}

