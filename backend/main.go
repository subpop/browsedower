package main

import (
	"log"
	"net/http"
	"os"

	"github.com/browsedower/web/database"
	"github.com/browsedower/web/handlers"
	"github.com/browsedower/web/middleware"
	"github.com/browsedower/web/services"
	"github.com/browsedower/web/websocket"
	"github.com/gorilla/mux"
)

func main() {
	// Initialize database
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./browsedower.db"
	}

	if err := database.Initialize(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize push notification service
	if err := services.InitPushService(); err != nil {
		log.Printf("Warning: Failed to initialize push service: %v", err)
	}

	// Start background scheduler
	services.StartScheduler()

	// Initialize WebSocket hub
	websocket.InitHub()

	// Create router
	r := mux.NewRouter()

	// CORS middleware
	r.Use(middleware.CORS)

	// Public API routes (token auth for extension)
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/patterns", middleware.TokenAuth(handlers.GetPatterns)).Methods("GET", "OPTIONS")
	api.HandleFunc("/requests", middleware.TokenAuth(handlers.CreateRequest)).Methods("POST", "OPTIONS")
	api.HandleFunc("/heartbeat", middleware.TokenAuth(handlers.DeviceHeartbeat)).Methods("POST", "OPTIONS")
	api.HandleFunc("/uninstall", handlers.DeviceUninstall).Methods("GET", "POST", "OPTIONS")
	api.HandleFunc("/ws", handlers.HandleWebSocket).Methods("GET")

	// Auth routes
	api.HandleFunc("/auth/login", handlers.Login).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/logout", middleware.SessionAuth(handlers.Logout)).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/change-password", middleware.SessionAuth(handlers.ChangePassword)).Methods("POST", "OPTIONS")

	// Setup routes (public, only work when no users exist)
	api.HandleFunc("/setup/status", handlers.CheckSetupNeeded).Methods("GET", "OPTIONS")
	api.HandleFunc("/setup/create-user", handlers.SetupFirstUser).Methods("POST", "OPTIONS")

	// Admin routes (session auth)
	admin := api.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.SessionAuthMiddleware)

	// Requests management
	admin.HandleFunc("/requests", handlers.ListRequests).Methods("GET", "OPTIONS")
	admin.HandleFunc("/requests/{id}/approve", handlers.ApproveRequest).Methods("POST", "OPTIONS")
	admin.HandleFunc("/requests/{id}/deny", handlers.DenyRequest).Methods("POST", "OPTIONS")

	// Patterns management
	admin.HandleFunc("/patterns", handlers.ListAllPatterns).Methods("GET", "OPTIONS")
	admin.HandleFunc("/patterns", handlers.CreatePattern).Methods("POST", "OPTIONS")
	admin.HandleFunc("/patterns/{id}", handlers.UpdatePattern).Methods("PUT", "OPTIONS")
	admin.HandleFunc("/patterns/{id}", handlers.DeletePattern).Methods("DELETE", "OPTIONS")
	admin.HandleFunc("/patterns/{id}/toggle", handlers.TogglePattern).Methods("POST", "OPTIONS")

	// Devices management
	admin.HandleFunc("/devices", handlers.ListDevices).Methods("GET", "OPTIONS")
	admin.HandleFunc("/devices", handlers.CreateDevice).Methods("POST", "OPTIONS")
	admin.HandleFunc("/devices/{id}", handlers.DeleteDevice).Methods("DELETE", "OPTIONS")
	admin.HandleFunc("/devices/{id}/regenerate-token", handlers.RegenerateDeviceToken).Methods("POST", "OPTIONS")

	// Users management
	admin.HandleFunc("/users", handlers.ListUsers).Methods("GET", "OPTIONS")
	admin.HandleFunc("/users", handlers.CreateUser).Methods("POST", "OPTIONS")

	// Push notifications
	admin.HandleFunc("/push/vapid-key", handlers.GetVAPIDPublicKey).Methods("GET", "OPTIONS")
	admin.HandleFunc("/push/subscribe", handlers.SubscribePush).Methods("POST", "OPTIONS")
	admin.HandleFunc("/push/unsubscribe", handlers.UnsubscribePush).Methods("POST", "OPTIONS")
	admin.HandleFunc("/push/subscriptions", handlers.GetPushSubscriptions).Methods("GET", "OPTIONS")
	admin.HandleFunc("/notifications/prefs", handlers.GetNotificationPrefs).Methods("GET", "OPTIONS")
	admin.HandleFunc("/notifications/prefs", handlers.UpdateNotificationPrefs).Methods("PUT", "OPTIONS")

	// Serve static files for admin UI
	staticDir := "./static"
	r.PathPrefix("/admin").Handler(http.StripPrefix("/admin", http.FileServer(http.Dir(staticDir))))
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
