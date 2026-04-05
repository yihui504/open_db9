package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/open-db9/db9/internal/api/middleware"
	"github.com/open-db9/db9/internal/auth"
	"github.com/open-db9/db9/internal/models"
)

// AnonymousAuthManager is an interface that abstracts auth.Manager for anonymous operations
type AnonymousAuthManager interface {
	GenerateAnonymousToken(tenantID string) (string, error)
	GenerateToken(userID int, username string, expiresIn int) (string, error)
}

// AnonymousRegisterHandler handles POST /api/v1/auth/anonymous
func AnonymousRegisterHandler(w http.ResponseWriter, r *http.Request, authManager AnonymousAuthManager) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req models.AnonymousRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate session_id
	if req.SessionID == "" {
		respondWithError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	// Get database connection
	db, err := GetDefaultDatabase()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	ctx := r.Context()

	// Check if session_id already exists (idempotency)
	var existingTenantID string
	var existingCapabilities []byte
	var existingExpiresAt time.Time
	err = db.QueryRow(ctx,
		"SELECT tenant_id, capabilities, expires_at FROM anonymous_accounts WHERE session_id = $1",
		req.SessionID).Scan(&existingTenantID, &existingCapabilities, &existingExpiresAt)

	if err == nil {
		// Session exists - return existing token (idempotent)
		token, err := authManager.GenerateAnonymousToken(existingTenantID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		respondWithJSON(w, http.StatusOK, Response{
			Success: true,
			Data: models.AnonymousRegisterResponse{
				Token:        json.RawMessage(fmt.Sprintf(`"%s"`, token)),
				TenantID:     existingTenantID,
				SessionID:    req.SessionID,
				Capabilities: json.RawMessage(existingCapabilities),
				ExpiresAt:    existingExpiresAt,
			},
		})
		return
	}

	if err != pgx.ErrNoRows {
		// Unexpected error
		respondWithError(w, http.StatusInternalServerError, "Failed to query anonymous account")
		return
	}

	// Generate short ID for tenant name
	shortID := generateShortID()

	// Create new tenant with anonymous plan
	var tenantID string
	tenantName := fmt.Sprintf("anonymous-%s", shortID)
	err = db.QueryRow(ctx,
		"INSERT INTO tenants (name, slug, plan, created_at, updated_at) VALUES ($1, $2, 'anonymous', NOW(), NOW()) RETURNING id",
		tenantName, tenantName).Scan(&tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create tenant")
		return
	}

	// Create anonymous account record
	defaultCapabilities := `{"max_databases": 5}`
	err = db.Execute(ctx,
		"INSERT INTO anonymous_accounts (session_id, tenant_id, capabilities, expires_at, created_at, updated_at) VALUES ($1, $2, $3::jsonb, NOW() + INTERVAL '24 hours', NOW(), NOW())",
		req.SessionID, tenantID, defaultCapabilities)

	if err != nil {
		// Clean up tenant if anonymous account creation failed
		db.Execute(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
		respondWithError(w, http.StatusInternalServerError, "Failed to create anonymous account")
		return
	}

	// Generate JWT token for anonymous user
	token, err := authManager.GenerateAnonymousToken(tenantID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Get the created record for response
	var capabilities []byte
	var expiresAt time.Time
	err = db.QueryRow(ctx,
		"SELECT capabilities, expires_at FROM anonymous_accounts WHERE session_id = $1",
		req.SessionID).Scan(&capabilities, &expiresAt)

	if err != nil {
		// Use defaults if query fails
		capabilities = []byte(defaultCapabilities)
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	respondWithJSON(w, http.StatusCreated, Response{
		Success: true,
		Data: models.AnonymousRegisterResponse{
			Token:        json.RawMessage(fmt.Sprintf(`"%s"`, token)),
			TenantID:     tenantID,
			SessionID:    req.SessionID,
			Capabilities: json.RawMessage(capabilities),
			ExpiresAt:    expiresAt,
		},
	})
}

// ClaimAccountHandler handles POST /api/v1/auth/claim
func ClaimAccountHandler(w http.ResponseWriter, r *http.Request, authManager AnonymousAuthManager) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user ID from context
	userID, ok := middleware.GetUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Parse request body
	var req models.ClaimAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate session_id
	if req.SessionID == "" {
		respondWithError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	// Get database connection
	db, err := GetDefaultDatabase()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	ctx := r.Context()

	// Find anonymous account by session_id
	var tenantID string
	var capabilities []byte
	err = db.QueryRow(ctx,
		"SELECT tenant_id, capabilities FROM anonymous_accounts WHERE session_id = $1",
		req.SessionID).Scan(&tenantID, &capabilities)

	if err == pgx.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Anonymous account not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to query anonymous account")
		return
	}

	// Update tenant plan from anonymous to free
	err = db.Execute(ctx,
		"UPDATE tenants SET plan = 'free', updated_at = NOW() WHERE id = $1 AND plan = 'anonymous'",
		tenantID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update tenant")
		return
	}

	// Delete anonymous account record
	err = db.Execute(ctx,
		"DELETE FROM anonymous_accounts WHERE session_id = $1",
		req.SessionID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete anonymous account")
		return
	}

	// Get username from context for new token generation
	username, _ := middleware.GetUsername(r)

	// Generate new full-permissions JWT token
	newToken, err := authManager.GenerateToken(userID, username, auth.DefaultTokenExpiry)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate new token")
		return
	}

	respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data: models.ClaimAccountResponse{
			Token:            json.RawMessage(fmt.Sprintf(`"%s"`, newToken)),
			Message:          "Account claimed successfully",
			PreviousTenantID: tenantID,
		},
	})
}

// generateShortID generates a random 8-character alphanumeric string
func generateShortID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}
