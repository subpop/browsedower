package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/browsedower/web/models"
)

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UsersResponse struct {
	Users []models.User `json:"users"`
}

// ListUsers returns all admin users (admin API)
func ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := models.ListUsers()
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	if users == nil {
		users = []models.User{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UsersResponse{Users: users})
}

// CreateUser creates a new admin user (admin API)
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	user, err := models.CreateUser(req.Username, req.Password)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

