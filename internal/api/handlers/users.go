package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/open-db9/db9/internal/api/middleware"
	"github.com/open-db9/db9/internal/database"
	"github.com/open-db9/db9/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// CreateUserHandler handles POST /api/v1/users
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Username == "" {
		respondWithError(w, http.StatusBadRequest, "Username is required")
		return
	}
	if req.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required")
		return
	}
	if req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Password is required")
		return
	}

	// Set default role
	if req.Role == "" {
		req.Role = "user"
	}

	// Get database from registry
	db, err := GetDefaultDatabase()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Check if username already exists
	var existingID int
	err = db.QueryRow(r.Context(), "SELECT id FROM users WHERE username = $1", req.Username).Scan(&existingID)
	if err != pgx.ErrNoRows {
		respondWithError(w, http.StatusConflict, "Username already exists")
		return
	}

	// Check if email already exists
	err = db.QueryRow(r.Context(), "SELECT id FROM users WHERE email = $1", req.Email).Scan(&existingID)
	if err != pgx.ErrNoRows {
		respondWithError(w, http.StatusConflict, "Email already exists")
		return
	}

	// Create user
	var userID int
	var createdAt, updatedAt time.Time
	err = db.QueryRow(r.Context(),
		"INSERT INTO users (username, email, password_hash, role, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id, created_at, updated_at",
		req.Username, req.Email, string(hashedPassword), req.Role).Scan(&userID, &createdAt, &updatedAt)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	user := models.User{
		ID:        userID,
		Username:  req.Username,
		Email:     req.Email,
		Role:      req.Role,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	respondWithJSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    user,
	})
}

// GetUserHandler handles GET /api/v1/users/:id
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from URL
	userID, err := extractIDFromURL(r.URL.Path, "/api/v1/users/")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get database
	db, err := GetDefaultDatabase()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	// Query user
	var user models.User
	err = db.QueryRow(r.Context(),
		"SELECT id, username, email, role, created_at, updated_at FROM users WHERE id = $1",
		userID).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    user,
	})
}

// ListUsersHandler handles GET /api/v1/users
func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse pagination parameters
	page := 1
	limit := 100

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Get database
	db, err := GetDefaultDatabase()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	// Query users
	rows, err := db.Query(r.Context(),
		"SELECT id, username, email, role, created_at, updated_at FROM users ORDER BY id LIMIT $1 OFFSET $2",
		limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to list users")
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt); err != nil {
			continue
		}
		users = append(users, user)
	}

	// Get total count
	var total int
	err = db.QueryRow(r.Context(), "SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		total = len(users)
	}

	response := models.ListUsersResponse{
		Users: users,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    response,
	})
}

// UpdateUserHandler handles PUT /api/v1/users/:id
func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID
	userID, err := extractIDFromURL(r.URL.Path, "/api/v1/users/")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Parse request body
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get database
	db, err := GetDefaultDatabase()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	// Check if user exists
	var existingUser models.User
	err = db.QueryRow(r.Context(),
		"SELECT id, username, email, role, created_at, updated_at FROM users WHERE id = $1",
		userID).Scan(&existingUser.ID, &existingUser.Username, &existingUser.Email, &existingUser.Role, &existingUser.CreatedAt, &existingUser.UpdatedAt)
	if err == pgx.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Email != "" && req.Email != existingUser.Email {
		// Check if email is already taken
		var existingID int
		err = db.QueryRow(r.Context(), "SELECT id FROM users WHERE email = $1 AND id != $2", req.Email, userID).Scan(&existingID)
		if err != pgx.ErrNoRows {
			respondWithError(w, http.StatusConflict, "Email already exists")
			return
		}
		updates = append(updates, fmt.Sprintf("email = $%d", argPos))
		args = append(args, req.Email)
		argPos++
		existingUser.Email = req.Email
	}

	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}
		updates = append(updates, fmt.Sprintf("password_hash = $%d", argPos))
		args = append(args, string(hashedPassword))
		argPos++
	}

	if req.Role != "" && req.Role != existingUser.Role {
		updates = append(updates, fmt.Sprintf("role = $%d", argPos))
		args = append(args, req.Role)
		argPos++
		existingUser.Role = req.Role
	}

	if len(updates) == 0 {
		respondWithJSON(w, http.StatusOK, Response{
			Success: true,
			Data:    existingUser,
		})
		return
	}

	// Add updated_at and user ID
	updates = append(updates, fmt.Sprintf("updated_at = NOW()"))
	args = append(args, userID)

	// Execute update
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", fmt.Sprintf("%s", updates), argPos)
	err = db.Execute(r.Context(), query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	// Fetch updated user
	err = db.QueryRow(r.Context(),
		"SELECT id, username, email, role, created_at, updated_at FROM users WHERE id = $1",
		userID).Scan(&existingUser.ID, &existingUser.Username, &existingUser.Email, &existingUser.Role, &existingUser.CreatedAt, &existingUser.UpdatedAt)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get updated user")
		return
	}

	respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    existingUser,
	})
}

// DeleteUserHandler handles DELETE /api/v1/users/:id
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID
	userID, err := extractIDFromURL(r.URL.Path, "/api/v1/users/")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Check admin role (from context)
	role, ok := GetRole(r)
	if !ok || role != "admin" {
		respondWithError(w, http.StatusForbidden, "Admin access required")
		return
	}

	// Get database
	db, err := GetDefaultDatabase()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	// Check if user exists
	var exists bool
	err = db.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil || !exists {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Delete user
	err = db.Execute(r.Context(), "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "User deleted successfully",
	})
}

// extractIDFromURL extracts ID from URL path
func extractIDFromURL(path, prefix string) (int, error) {
	if !checkPrefix(path, prefix) {
		return 0, fmt.Errorf("invalid URL format")
	}
	rest := trimPrefix(path, prefix)
	idStr := rest
	if idx := indexOfSlash(rest); idx != -1 {
		idStr = rest[:idx]
	}
	return strconv.Atoi(idStr)
}

// Helper functions for URL parsing
func checkPrefix(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

func trimPrefix(path, prefix string) string {
	if len(path) > len(prefix) {
		return path[len(prefix):]
	}
	return ""
}

func indexOfSlash(s string) int {
	for i, c := range s {
		if c == '/' {
			return i
		}
	}
	return -1
}

// Helper functions for HTTP responses
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   message,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

// GetDefaultDatabase returns the default database connection pool from the registry
func GetDefaultDatabase() (*database.ConnectionPool, error) {
	db, err := GetDatabase(1)
	if err != nil {
		return nil, err
	}
	return db.GetPool(), nil
}

// GetRole retrieves the user role from the request context
func GetRole(r *http.Request) (string, bool) {
	return middleware.GetRole(r)
}
