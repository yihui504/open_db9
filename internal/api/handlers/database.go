package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type CreateDatabaseRequest struct {
	Name   string `json:"name"`
	Engine string `json:"engine"`
}

type DatabaseResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Engine    string    `json:"engine"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type DatabaseListResponse struct {
	Databases []DatabaseResponse `json:"databases"`
	Count     int                `json:"count"`
}

func CreateDatabaseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var req CreateDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Database name is required", http.StatusBadRequest)
		return
	}

	if req.Engine == "" {
		req.Engine = "postgresql"
	}

	dbID := uuid.New()
	now := time.Now()

	resp := Response{
		Success: true,
		Data: DatabaseResponse{
			ID:        dbID,
			Name:      req.Name,
			Engine:    req.Engine,
			Status:    "creating",
			CreatedAt: now,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func ListDatabasesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := Response{
		Success: true,
		Data: DatabaseListResponse{
			Databases: []DatabaseResponse{
				{
					ID:        uuid.MustParse("b0000000-0000-0000-0000-000000000001"),
					Name:      "default-db",
					Engine:    "postgresql",
					Status:    "active",
					CreatedAt: time.Now().Add(-24 * time.Hour),
				},
			},
			Count: 1,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
