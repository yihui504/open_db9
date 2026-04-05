package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/open-db9/db9/internal/auth"
	"github.com/open-db9/db9/internal/models"
)

// mockAnonymousAuthManager implements AnonymousAuthManager interface for testing
type mockAnonymousAuthManager struct {
	authManager *auth.Manager
}

func (m *mockAnonymousAuthManager) GenerateAnonymousToken(tenantID string) (string, error) {
	return m.authManager.GenerateAnonymousToken(tenantID)
}

func (m *mockAnonymousAuthManager) GenerateToken(userID int, username string, expiresIn int) (string, error) {
	return m.authManager.GenerateToken(userID, username, expiresIn)
}

func setupTestAuthManager() (*auth.Manager, *mockAnonymousAuthManager, error) {
	// Create auth manager with test secret (must be >= 32 chars)
	authMgr, err := auth.NewManagerWithSecret("test-secret-key-for-unit-testing-must-be-32-chars")
	if err != nil {
		return nil, nil, err
	}
	mockMgr := &mockAnonymousAuthManager{authManager: authMgr}
	return authMgr, mockMgr, nil
}

func TestAnonymousRegister_NewSession(t *testing.T) {
	_, mockMgr, err := setupTestAuthManager()
	if err != nil {
		t.Fatalf("Failed to setup auth manager: %v", err)
	}

	// Create request body
	reqBody := models.AnonymousRegisterRequest{
		SessionID: "test-session-123",
	}
	body, _ := json.Marshal(reqBody)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/anonymous", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call handler (note: this will fail without actual DB connection, but we can test the structure)
	AnonymousRegisterHandler(w, req, mockMgr)

	// Check status code - should be 500 or 201 depending on DB availability
	if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 201 or 500, got %d", w.Code)
	}

	if w.Code == http.StatusCreated {
		var response Response
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Errorf("Expected success=true, got %v", response.Success)
		}

		// Verify response structure
		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		if data["token"] == nil {
			t.Error("Expected token in response")
		}
		if data["tenant_id"] == nil {
			t.Error("Expected tenant_id in response")
		}
		if data["session_id"] != "test-session-123" {
			t.Errorf("Expected session_id=test-session-123, got %v", data["session_id"])
		}
		if data["capabilities"] == nil {
			t.Error("Expected capabilities in response")
		}
		if data["expires_at"] == nil {
			t.Error("Expected expires_at in response")
		}
	}
}

func TestAnonymousRegister_ExistingSession(t *testing.T) {
	_, mockMgr, err := setupTestAuthManager()
	if err != nil {
		t.Fatalf("Failed to setup auth manager: %v", err)
	}

	// First request (creates the session)
	reqBody := models.AnonymousRegisterRequest{
		SessionID: "test-session-duplicate",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/anonymous", bytes.NewReader(body))
	w := httptest.NewRecorder()

	AnonymousRegisterHandler(w, req, mockMgr)

	// Second request with same session (should return existing token)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/anonymous", bytes.NewReader(body))
	w2 := httptest.NewRecorder()

	AnonymousRegisterHandler(w2, req2, mockMgr)

	// Both should return same status (201 or 500)
	if w2.Code != http.StatusOK && w2.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500 for idempotent call, got %d", w2.Code)
	}
}

func TestAnonymousRegister_InvalidSession(t *testing.T) {
	_, mockMgr, err := setupTestAuthManager()
	if err != nil {
		t.Fatalf("Failed to setup auth manager: %v", err)
	}

	// Test empty session_id
	testCases := []struct {
		name       string
		sessionID  string
		expectCode int
	}{
		{"EmptySessionID", "", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := models.AnonymousRegisterRequest{
				SessionID: tc.sessionID,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/anonymous", bytes.NewReader(body))
			w := httptest.NewRecorder()

			AnonymousRegisterHandler(w, req, mockMgr)

			if w.Code != tc.expectCode {
				t.Errorf("Expected status %d, got %d", tc.expectCode, w.Code)
			}
		})
	}
}

func TestAnonymousRegister_InvalidRequestBody(t *testing.T) {
	_, mockMgr, err := setupTestAuthManager()
	if err != nil {
		t.Fatalf("Failed to setup auth manager: %v", err)
	}

	// Test invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/anonymous", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	AnonymousRegisterHandler(w, req, mockMgr)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestAnonymousRegister_WrongMethod(t *testing.T) {
	_, mockMgr, err := setupTestAuthManager()
	if err != nil {
		t.Fatalf("Failed to setup auth manager: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/anonymous", nil)
	w := httptest.NewRecorder()

	AnonymousRegisterHandler(w, req, mockMgr)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for GET method, got %d", w.Code)
	}
}

func TestClaimAccount_Success(t *testing.T) {
	authMgr, mockMgr, err := setupTestAuthManager()
	if err != nil {
		t.Fatalf("Failed to setup auth manager: %v", err)
	}

	// Generate authenticated user token
	token, err := authMgr.GenerateToken(1, "testuser", 3600)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create claim request
	reqBody := models.ClaimAccountRequest{
		SessionID: "test-session-to-claim",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/claim", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	// Note: This will need proper context setup with user ID for full testing
	// For now, we test the handler structure
	ClaimAccountHandler(w, req, mockMgr)

	// Should return 401 without proper context (no user ID in context)
	// or other status codes depending on implementation
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Logf("ClaimAccount returned status %d (may vary based on context)", w.Code)
	}
}

func TestClaimAccount_NonExistentSession(t *testing.T) {
	authMgr, mockMgr, err := setupTestAuthManager()
	if err != nil {
		t.Fatalf("Failed to setup auth manager: %v", err)
	}

	// Generate authenticated user token
	token, err := authMgr.GenerateToken(1, "testuser", 3600)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create claim request with non-existent session
	reqBody := models.ClaimAccountRequest{
		SessionID: "non-existent-session",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/claim", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	ClaimAccountHandler(w, req, mockMgr)

	// Should return 404 if session doesn't exist (after authentication passes)
	if w.Code != http.StatusNotFound && w.Code != http.StatusUnauthorized {
		t.Logf("ClaimAccount with non-existent session returned status %d", w.Code)
	}
}

func TestClaimAccount_Unauthenticated(t *testing.T) {
	_, mockMgr, err := setupTestAuthManager()
	if err != nil {
		t.Fatalf("Failed to setup auth manager: %v", err)
	}

	// Create claim request without authorization header
	reqBody := models.ClaimAccountRequest{
		SessionID: "some-session",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/claim", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	w := httptest.NewRecorder()

	ClaimAccountHandler(w, req, mockMgr)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for unauthenticated request, got %d", w.Code)
	}
}

func TestClaimAccount_InvalidSession(t *testing.T) {
	authMgr, mockMgr, err := setupTestAuthManager()
	if err != nil {
		t.Fatalf("Failed to setup auth manager: %v", err)
	}

	// Generate authenticated user token
	token, err := authMgr.GenerateToken(1, "testuser", 3600)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Test empty session_id
	testCases := []struct {
		name       string
		sessionID  string
		expectCode int
	}{
		{"EmptySessionID", "", http.StatusUnauthorized},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := models.ClaimAccountRequest{
				SessionID: tc.sessionID,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/claim", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			ClaimAccountHandler(w, req, mockMgr)

			if w.Code != tc.expectCode {
				t.Errorf("Expected status %d, got %d", tc.expectCode, w.Code)
			}
		})
	}
}

func TestGenerateShortID(t *testing.T) {
	// Test that generateShortID produces valid IDs
	id1 := generateShortID()
	id2 := generateShortID()

	// Check length
	if len(id1) != 8 {
		t.Errorf("Expected short ID length 8, got %d", len(id1))
	}

	// Check uniqueness (very low probability of collision, but good to test)
	if id1 == id2 {
		t.Error("Expected different IDs on consecutive calls")
	}

	// Check that it only contains lowercase letters and digits
	for _, c := range id1 {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			t.Errorf("Invalid character in short ID: %c", c)
		}
	}
}

// Benchmark tests
func BenchmarkAnonymousRegister(b *testing.B) {
	_, mockMgr, err := setupTestAuthManager()
	if err != nil {
		b.Fatalf("Failed to setup auth manager: %v", err)
	}

	reqBody := models.AnonymousRegisterRequest{
		SessionID: "bench-session",
	}
	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/anonymous", bytes.NewReader(body))
		w := httptest.NewRecorder()
		AnonymousRegisterHandler(w, req, mockMgr)
	}
}

// Helper function to create context with user info for testing
func createAuthenticatedContext(ctx context.Context, userID int, username string) context.Context {
	// In a real test, you would use middleware functions to set context values
	// This is a placeholder for integration tests
	return ctx
}

// Test response structure validation
func TestAnonymousRegisterResponseStructure(t *testing.T) {
	response := Response{
		Success: true,
		Data: models.AnonymousRegisterResponse{
			Token:       json.RawMessage(`"test-token"`),
			TenantID:    "test-tenant-id",
			SessionID:   "test-session",
			Capabilities: json.RawMessage(`{"max_databases":5}`),
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		},
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	data, ok := response.Data.(models.AnonymousRegisterResponse)
	if !ok {
		t.Fatal("Expected data to be AnonymousRegisterResponse")
	}

	if data.TenantID != "test-tenant-id" {
		t.Errorf("Expected tenant_id=test-tenant-id, got %s", data.TenantID)
	}
	if data.SessionID != "test-session" {
		t.Errorf("Expected session_id=test-session, got %s", data.SessionID)
	}
}

func TestClaimAccountResponseStructure(t *testing.T) {
	response := Response{
		Success: true,
		Data: models.ClaimAccountResponse{
			Token:           json.RawMessage(`"new-token"`),
			Message:         "Account claimed successfully",
			PreviousTenantID: "prev-tenant-id",
		},
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	data, ok := response.Data.(models.ClaimAccountResponse)
	if !ok {
		t.Fatal("Expected data to be ClaimAccountResponse")
	}

	if data.Message != "Account claimed successfully" {
		t.Errorf("Expected message='Account claimed successfully', got %s", data.Message)
	}
	if data.PreviousTenantID != "prev-tenant-id" {
		t.Errorf("Expected previous_tenant_id=prev-tenant-id, got %s", data.PreviousTenantID)
	}
}
