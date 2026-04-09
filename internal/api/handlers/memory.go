package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type MemoryStoreRequest struct {
	AgentID        string   `json:"agent_id"`
	SessionID      string   `json:"session_id,omitempty"`
	MemoryType     string   `json:"memory_type"`
	Content        string   `json:"content"`
	Summary        string   `json:"summary,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	ImportanceScore float64  `json:"importance_score,omitempty"`
}

type MemoryRecallRequest struct {
	Query  string  `json:"query"`
	TopK   int     `json:"top_k,omitempty"`
	Filter struct {
		AgentID    string   `json:"agent_id,omitempty"`
		MemoryType string   `json:"memory_type,omitempty"`
		Tags       []string `json:"tags,omitempty"`
	} `json:"filter,omitempty"`
}

func StoreMemoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 6 || parts[5] != "memories" {
		http.Error(w, "Invalid URL format. Expected: /api/v1/databases/:id/memories", http.StatusBadRequest)
		return
	}

	dbIDStr := parts[4]
	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database ID: %s", dbIDStr), http.StatusBadRequest)
		return
	}

	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req MemoryStoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	if req.MemoryType == "" {
		req.MemoryType = "fact"
	}

	if req.ImportanceScore <= 0 {
		req.ImportanceScore = 0.5
	}

	if req.AgentID == "" {
		req.AgentID = "default"
	}

	pool := manager.GetPool()
	ctx := r.Context()

	var memoryID string
	err = pool.QueryRow(ctx, `
		INSERT INTO agent_memories (agent_id, session_id, memory_type, content, summary, tags, importance_score)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`, req.AgentID, req.SessionID, req.MemoryType,
		req.Content, req.Summary, req.Tags, req.ImportanceScore).Scan(&memoryID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store memory: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"id":         memoryID,
			"agent_id":   req.AgentID,
			"memory_type": req.MemoryType,
		},
	})
}

func RecallMemoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 7 || parts[5] != "memories" || parts[6] != "recall" {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	dbID, _ := strconv.Atoi(parts[4])
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req MemoryRecallRequest
	json.NewDecoder(r.Body).Decode(&req)

	topK := req.TopK
	if topK <= 0 || topK > 50 {
		topK = 10
	}

	pool := manager.GetPool()
	ctx := r.Context()

	query := `
		SELECT id, agent_id, session_id, memory_type, content, summary, 
		       tags, importance_score, access_count, created_at
		FROM agent_memories
		WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if req.Filter.AgentID != "" {
		query += fmt.Sprintf(" AND agent_id = $%d", argIdx)
		args = append(args, req.Filter.AgentID)
		argIdx++
	}
	if req.Filter.MemoryType != "" {
		query += fmt.Sprintf(" AND memory_type = $%d", argIdx)
		args = append(args, req.Filter.MemoryType)
		argIdx++
	}
	if len(req.Filter.Tags) > 0 {
		query += fmt.Sprintf(" && tags && $%d", argIdx)
		args = append(args, req.Filter.Tags)
		argIdx++
	}
	if req.Query != "" {
		query += fmt.Sprintf(" AND (content ILIKE $%d OR summary ILIKE $%d)", argIdx, argIdx+1)
		args = append(args, "%"+req.Query+"%", "%"+req.Query+"%")
		argIdx += 2
	}

	query += fmt.Sprintf(" ORDER BY importance_score DESC, created_at DESC LIMIT $%d", argIdx)
	args = append(args, topK)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Recall failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	memories := []map[string]interface{}{}
	for rows.Next() {
		var id, agentID, memType, content string
		var summary sql.NullString
		var tags []string
		var importance float64
		var accessCount int
		var createdAt time.Time
		var sessionID sql.NullString

		rows.Scan(&id, &agentID, &sessionID, &memType, &content, &summary, &tags, &importance, &accessCount, &createdAt)

		m := map[string]interface{}{
			"id":               id,
			"agent_id":         agentID,
			"memory_type":      memType,
			"content":          content,
			"importance_score": importance,
			"access_count":     accessCount,
			"created_at":       createdAt.Format(time.RFC3339),
		}
		if summary.Valid { m["summary"] = summary.String }
		if sessionID.Valid { m["session_id"] = sessionID.String }
		if len(tags) > 0 { m["tags"] = tags }
		memories = append(memories, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]interface{}{
			"count":    len(memories),
			"memories": memories,
		},
	})
}

func ListMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 6 || parts[5] != "memories" {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	dbID, _ := strconv.Atoi(parts[4])
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	q := r.URL.Query()
	agentID := q.Get("agent_id")
	memType := q.Get("memory_type")
	tags := q.Get("tags")

	pool := manager.GetPool()
	ctx := r.Context()

	query := `SELECT id, agent_id, memory_type, content, summary, tags, importance_score, created_at FROM agent_memories WHERE 1=1`
	args := []interface{}{}
	idx := 1

	if agentID != "" {
		query += fmt.Sprintf(" AND agent_id = $%d", idx)
		args = append(args, agentID)
		idx++
	}
	if memType != "" {
		query += fmt.Sprintf(" AND memory_type = $%d", idx)
		args = append(args, memType)
		idx++
	}
	if tags != "" {
		query += fmt.Sprintf(" && tags && $%d", idx)
		args = append(args, strings.Split(tags, ","))
		idx++
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("List failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	memories := []map[string]interface{}{}
	for rows.Next() {
		var id, agentID, memType, content string
		var summary sql.NullString
		var tags []string
		var importance float64
		var createdAt time.Time

		rows.Scan(&id, &agentID, &memType, &content, &summary, &tags, &importance, &createdAt)
		m := map[string]interface{}{
			"id": id, "agent_id": agentID, "memory_type": memType,
			"content": content, "importance_score": importance, "created_at": createdAt.Format(time.RFC3339),
		}
		if summary.Valid { m["summary"] = summary.String }
		if len(tags) > 0 { m["tags"] = tags }
		memories = append(memories, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Success: true, Data: map[string]interface{}{"count": len(memories), "memories": memories}})
}

func DeleteMemoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 7 || parts[5] != "memories" {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	dbID, _ := strconv.Atoi(parts[4])
	memoryID := parts[6]

	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	pool := manager.GetPool()
	var deletedID string
	err = pool.QueryRow(r.Context(), "DELETE FROM agent_memories WHERE id = $1 RETURNING id", memoryID).Scan(&deletedID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Delete failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err == nil && deletedID != "" {
		json.NewEncoder(w).Encode(Response{Success: true, Data: map[string]string{"deleted": deletedID}})
	} else if strings.Contains(err.Error(), "no rows") {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Memory not found"})
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Memory not found"})
	}
}
