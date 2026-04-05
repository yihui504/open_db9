package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RAGHandler handles RAG-related requests
type RAGHandler struct {
	db *sql.DB
}

// NewRAGHandler creates a new RAG handler
func NewRAGHandler(db *sql.DB) *RAGHandler {
	return &RAGHandler{db: db}
}

// CreateDocumentRequest represents a request to create a document record
type CreateDocumentRequest struct {
	Title    string                 `json:"title"`
	Filename string                 `json:"filename"`
	FS9Key   string                 `json:"fs9_key"`
	Metadata map[string]interface{} `json:"metadata"`
}

// CreateDocument creates a new document record
func (h *RAGHandler) CreateDocument(c *gin.Context) {
	databaseID := c.Param("id")
	tenantID := c.GetString("tenant_id") // From JWT middleware

	var req CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate database belongs to tenant
	if !h.databaseBelongsToTenant(databaseID, tenantID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Create document record
	query := `
		INSERT INTO rag_documents (tenant_id, database_id, fs9_key, title, filename, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var documentID uuid.UUID
	err := h.db.QueryRow(
		query,
		uuid.MustParse(tenantID),
		uuid.MustParse(databaseID),
		req.FS9Key,
		req.Title,
		req.Filename,
		req.Metadata,
	).Scan(&documentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": documentID.String()})
}

// ListDocuments lists all documents for a database
func (h *RAGHandler) ListDocuments(c *gin.Context) {
	databaseID := c.Param("id")
	tenantID := c.GetString("tenant_id")

	// Set RLS context
	_, err := h.db.Exec("SELECT set_rag_tenant_id($1)", uuid.MustParse(tenantID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	query := `
		SELECT id, title, filename, total_chunks, created_at
		FROM rag_documents
		WHERE database_id = $1
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query, uuid.MustParse(databaseID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	documents := []map[string]interface{}{}
	for rows.Next() {
		var id, title, filename string
		var chunks int
		var createdAt time.Time
		if err := rows.Scan(&id, &title, &filename, &chunks, &createdAt); err != nil {
			continue
		}
		documents = append(documents, map[string]interface{}{
			"id":           id,
			"title":        title,
			"filename":     filename,
			"total_chunks": chunks,
			"created_at":   createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"documents": documents, "total": len(documents)})
}

// DeleteDocument deletes a document and its chunks
func (h *RAGHandler) DeleteDocument(c *gin.Context) {
	documentID := c.Param("document_id")
	tenantID := c.GetString("tenant_id")

	// Set RLS context
	_, err := h.db.Exec("SELECT set_rag_tenant_id($1)", uuid.MustParse(tenantID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Delete will cascade to chunks due to ON DELETE CASCADE
	result, err := h.db.Exec("DELETE FROM rag_documents WHERE id = $1", uuid.MustParse(documentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// UpdateDocumentChunkCount updates the chunk count for a document
func (h *RAGHandler) UpdateDocumentChunkCount(c *gin.Context) {
	databaseID := c.Param("id")
	documentID := c.Param("document_id")
	tenantID := c.GetString("tenant_id")

	var req struct {
		TotalChunks int `json:"total_chunks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set RLS context
	_, err := h.db.Exec("SELECT set_rag_tenant_id($1)", uuid.MustParse(tenantID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	query := `
		UPDATE rag_documents
		SET total_chunks = $1, updated_at = NOW()
		WHERE id = $2 AND database_id = $3
	`

	_, err = h.db.Exec(query, req.TotalChunks, uuid.MustParse(documentID), uuid.MustParse(databaseID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *RAGHandler) databaseBelongsToTenant(databaseID, tenantID string) bool {
	var exists bool
	err := h.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM databases WHERE id = $1 AND tenant_id = $2)",
		uuid.MustParse(databaseID),
		uuid.MustParse(tenantID),
	).Scan(&exists)
	return err == nil && exists
}

// CreateDocumentHandler handles document creation requests
func CreateDocumentHandler(w http.ResponseWriter, r *http.Request) {
	// This handler requires a database connection
	// For now, return not implemented - the actual RAG ingestion
	// is handled by the Python FastAPI service
	http.Error(w, "Document creation handled by RAG Python service", http.StatusNotImplemented)
}

// ListDocumentsHandler lists all documents for a database
func ListDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	// This handler requires a database connection
	// For now, return not implemented
	http.Error(w, "Document listing not yet implemented", http.StatusNotImplemented)
}

// GetDocumentHandler gets a specific document
func GetDocumentHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Document retrieval not yet implemented", http.StatusNotImplemented)
}

// DeleteDocumentHandler deletes a document
func DeleteDocumentHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Document deletion not yet implemented", http.StatusNotImplemented)
}

// UpdateDocumentChunkCountHandler updates document chunk count
func UpdateDocumentChunkCountHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Document update not yet implemented", http.StatusNotImplemented)
}
