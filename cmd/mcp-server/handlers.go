package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// registerTools registers all MCP tools with the server
func registerTools(s *mcpserver.MCPServer) {
	// SQL execution tool
	s.AddTool(mcp.NewTool("execute_sql",
		mcp.WithDescription("Execute a SQL query on a specific database. Returns the result set as JSON. Use this for SELECT, INSERT, UPDATE, DELETE, and DDL operations."),
		mcp.WithString("database_id",
			mcp.Required(),
			mcp.Description("The database ID (numeric) to execute the query on"),
		),
		mcp.WithString("sql",
			mcp.Required(),
			mcp.Description("SQL query to execute"),
		),
	), handleExecuteSQL)

	// List databases tool
	s.AddTool(mcp.NewTool("list_databases",
		mcp.WithDescription("List all databases accessible to the current user. Returns database IDs, names, and connection information."),
	), handleListDatabases)

	// Create database tool
	s.AddTool(mcp.NewTool("create_database",
		mcp.WithDescription("Create a new database instance in the system."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the database"),
		),
		mcp.WithString("engine",
			mcp.Description("Database engine type (default: postgresql)"),
		),
	), handleCreateDatabase)

	// List files tool
	s.AddTool(mcp.NewTool("list_files",
		mcp.WithDescription("List files stored in a database's file storage (FS9). Supports path filtering and pagination."),
		mcp.WithString("database_id",
			mcp.Required(),
			mcp.Description("The database ID to list files from"),
		),
		mcp.WithString("path",
			mcp.Description("Optional path filter (e.g., '/documents/' for prefix matching)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of files to return (default: 100, max: 1000)"),
		),
	), handleListFiles)

	// Upload file tool
	s.AddTool(mcp.NewTool("upload_file",
		mcp.WithDescription("Upload a file to a database's file storage (FS9). The file content should be provided as base64 encoded string or text content."),
		mcp.WithString("database_id",
			mcp.Required(),
			mcp.Description("The database ID to upload the file to"),
		),
		mcp.WithString("filename",
			mcp.Required(),
			mcp.Description("Name of the file to upload"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("File content (text or base64 encoded binary)"),
		),
		mcp.WithString("path",
			mcp.Description("Optional path where to store the file (default: /)"),
		),
		mcp.WithString("content_type",
			mcp.Description("MIME type of the file (auto-detected if not provided)"),
		),
	), handleUploadFile)

	// Download file tool
	s.AddTool(mcp.NewTool("download_file",
		mcp.WithDescription("Download a file from a database's file storage (FS9). Returns the file content and metadata."),
		mcp.WithString("database_id",
			mcp.Required(),
			mcp.Description("The database ID to download the file from"),
		),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path of the file to download"),
		),
	), handleDownloadFile)

	// Query RAG tool
	s.AddTool(mcp.NewTool("query_rag",
		mcp.WithDescription("Query the RAG (Retrieval-Augmented Generation) system with natural language questions. This searches through indexed documents and returns relevant context along with answers."),
		mcp.WithString("database_id",
			mcp.Required(),
			mcp.Description("The database ID to query RAG for"),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Natural language question to search for"),
		),
		mcp.WithNumber("top_k",
			mcp.Description("Number of relevant chunks to retrieve (default: 5)"),
		),
	), handleQueryRAG)

	// Get schema tool
	s.AddTool(mcp.NewTool("get_schema",
		mcp.WithDescription("Get detailed schema information for a specific table including columns, indexes, constraints, and triggers."),
		mcp.WithString("database_id",
			mcp.Required(),
			mcp.Description("The database ID to get schema information from"),
		),
		mcp.WithString("table",
			mcp.Required(),
			mcp.Description("Name of the table to get schema details for"),
		),
		mcp.WithString("schema",
			mcp.Description("Schema name (default: public)"),
		),
	), handleGetSchema)

	// Create snapshot tool
	s.AddTool(mcp.NewTool("create_snapshot",
		mcp.WithDescription("Create a database backup snapshot. This is an asynchronous operation that creates a point-in-time backup."),
		mcp.WithString("database_id",
			mcp.Required(),
			mcp.Description("The database ID to create a snapshot for"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Descriptive name for the snapshot"),
		),
	), handleCreateSnapshot)

	// List snapshots tool
	s.AddTool(mcp.NewTool("list_snapshots",
		mcp.WithDescription("List all snapshots for a database, showing their status, size, and creation time."),
		mcp.WithString("database_id",
			mcp.Required(),
			mcp.Description("The database ID to list snapshots for"),
		),
	), handleListSnapshots)

	// Restore snapshot tool
	s.AddTool(mcp.NewTool("restore_snapshot",
		mcp.WithDescription("Restore a database from a previous snapshot. This is an asynchronous operation that will overwrite the current database state."),
		mcp.WithString("database_id",
			mcp.Required(),
			mcp.Description("The database ID to restore"),
		),
		mcp.WithString("snapshot_id",
			mcp.Required(),
			mcp.Description("ID of the snapshot to restore from"),
		),
	), handleRestoreSnapshot)
}

// handleExecuteSQL handles SQL execution requests
func handleExecuteSQL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbID, err := request.RequireString("database_id")
	if err != nil {
		return mcp.NewToolResultError("database_id is required"), nil
	}

	sql, err := request.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError("sql is required"), nil
	}

	log.Printf("[execute_sql] Database: %s, Query: %s", dbID, truncateSQL(sql))

	resp, err := client.Do("POST", fmt.Sprintf("/api/v1/databases/%s/sql", dbID), map[string]string{
		"query": sql,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to execute SQL: %v", err)), nil
	}

	if !resp.Success && resp.Error != "" {
		return mcp.NewToolResultError(fmt.Sprintf("SQL execution error: %s", resp.Error)), nil
	}

	resultJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleListDatabases handles listing databases
func handleListDatabases(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := client.Do("GET", "/api/v1/databases/", nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list databases: %v", err)), nil
	}

	resultJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleCreateDatabase handles creating a new database
func handleCreateDatabase(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}

	engine := getStringArg(request, "engine", "postgresql")

	reqBody := map[string]interface{}{
		"name":   name,
		"engine": engine,
	}

	resp, err := client.Do("POST", "/api/v1/databases/", reqBody)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create database: %v", err)), nil
	}

	if !resp.Success && resp.Error != "" {
		return mcp.NewToolResultError(fmt.Sprintf("Database creation error: %s", resp.Error)), nil
	}

	resultJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Database created successfully:\n%s", string(resultJSON))), nil
}

// handleListFiles handles listing files in FS9 storage
func handleListFiles(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbID, err := request.RequireString("database_id")
	if err != nil {
		return mcp.NewToolResultError("database_id is required"), nil
	}

	path := getStringArg(request, "path", "")

	urlPath := fmt.Sprintf("/api/v1/databases/%s/files", dbID)
	if path != "" {
		urlPath += "?path=" + path
	}

	args := request.Params.Arguments
	argsMap, _ := args.(map[string]interface{})
	if limit, ok := argsMap["limit"]; ok && limit != nil {
		if strings.Contains(urlPath, "?") {
			urlPath += "&limit="
		} else {
			urlPath += "?limit="
		}
		urlPath += fmt.Sprintf("%.0f", limit.(float64))
	}

	resp, err := client.Do("GET", urlPath, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list files: %v", err)), nil
	}

	if !resp.Success && resp.Error != "" {
		return mcp.NewToolResultError(fmt.Sprintf("List files error: %s", resp.Error)), nil
	}

	resultJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleUploadFile handles file upload to FS9 storage
func handleUploadFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbID, err := request.RequireString("database_id")
	if err != nil {
		return mcp.NewToolResultError("database_id is required"), nil
	}

	filename, err := request.RequireString("filename")
	if err != nil {
		return mcp.NewToolResultError("filename is required"), nil
	}

	content, err := request.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("content is required"), nil
	}

	path := getStringArg(request, "path", "/"+filename)
	contentType := getStringArg(request, "content_type", "")

	// Note: For actual file uploads via MCP, we would need multipart form handling
	// For now, we'll send metadata and note that full implementation requires FS9 integration
	reqBody := map[string]interface{}{
		"filename":     filename,
		"content":      content,
		"path":         path,
		"content_type": contentType,
	}

	resp, err := client.Do("POST", fmt.Sprintf("/api/v1/databases/%s/files", dbID), reqBody)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to upload file: %v", err)), nil
	}

	if !resp.Success && resp.Error != "" {
		return mcp.NewToolResultError(fmt.Sprintf("Upload error: %s", resp.Error)), nil
	}

	resultJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("File uploaded successfully:\n%s", string(resultJSON))), nil
}

// handleDownloadFile handles file download from FS9 storage
func handleDownloadFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbID, err := request.RequireString("database_id")
	if err != nil {
		return mcp.NewToolResultError("database_id is required"), nil
	}

	filePath, err := request.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError("file_path is required"), nil
	}

	resp, err := client.Do("GET", fmt.Sprintf("/api/v1/databases/%s/files%s", dbID, filePath), nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to download file: %v", err)), nil
	}

	if !resp.Success && resp.Error != "" {
		return mcp.NewToolResultError(fmt.Sprintf("Download error: %s", resp.Error)), nil
	}

	resultJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleQueryRAG handles RAG queries
func handleQueryRAG(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbID, err := request.RequireString("database_id")
	if err != nil {
		return mcp.NewToolResultError("database_id is required"), nil
	}

	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query is required"), nil
	}

	topK := 5
	args := request.Params.Arguments
	argsMap, _ := args.(map[string]interface{})
	if tk, ok := argsMap["top_k"]; ok && tk != nil {
		topK = int(tk.(float64))
	}

	reqBody := map[string]interface{}{
		"database_id": dbID,
		"query":       query,
		"top_k":       topK,
	}

	// Query the RAG Python service
	resp, err := client.DoRAG("POST", "/rag/query", reqBody)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query RAG: %v", err)), nil
	}

	if !resp.Success && resp.Error != "" {
		return mcp.NewToolResultError(fmt.Sprintf("RAG query error: %s", resp.Error)), nil
	}

	resultJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGetSchema handles getting table schema details
func handleGetSchema(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbID, err := request.RequireString("database_id")
	if err != nil {
		return mcp.NewToolResultError("database_id is required"), nil
	}

	table, err := request.RequireString("table")
	if err != nil {
		return mcp.NewToolResultError("table is required"), nil
	}

	schema := getStringArg(request, "schema", "public")

	urlPath := fmt.Sprintf("/api/v1/databases/%s/schema/table?schema=%s&table=%s", dbID, schema, table)

	resp, err := client.Do("GET", urlPath, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get schema: %v", err)), nil
	}

	if !resp.Success && resp.Error != "" {
		return mcp.NewToolResultError(fmt.Sprintf("Schema query error: %s", resp.Error)), nil
	}

	resultJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleCreateSnapshot handles creating a database snapshot
func handleCreateSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbID, err := request.RequireString("database_id")
	if err != nil {
		return mcp.NewToolResultError("database_id is required"), nil
	}

	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}

	reqBody := map[string]string{
		"name": name,
	}

	resp, err := client.Do("POST", fmt.Sprintf("/api/v1/databases/%s/snapshots", dbID), reqBody)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create snapshot: %v", err)), nil
	}

	if !resp.Success && resp.Error != "" {
		return mcp.NewToolResultError(fmt.Sprintf("Snapshot creation error: %s", resp.Error)), nil
	}

	resultJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Snapshot creation initiated:\n%s", string(resultJSON))), nil
}

// handleListSnapshots handles listing database snapshots
func handleListSnapshots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbID, err := request.RequireString("database_id")
	if err != nil {
		return mcp.NewToolResultError("database_id is required"), nil
	}

	resp, err := client.Do("GET", fmt.Sprintf("/api/v1/databases/%s/snapshots", dbID), nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list snapshots: %v", err)), nil
	}

	if !resp.Success && resp.Error != "" {
		return mcp.NewToolResultError(fmt.Sprintf("List snapshots error: %s", resp.Error)), nil
	}

	resultJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleRestoreSnapshot handles restoring from a snapshot
func handleRestoreSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbID, err := request.RequireString("database_id")
	if err != nil {
		return mcp.NewToolResultError("database_id is required"), nil
	}

	snapshotID, err := request.RequireString("snapshot_id")
	if err != nil {
		return mcp.NewToolResultError("snapshot_id is required"), nil
	}

	resp, err := client.Do("POST", fmt.Sprintf("/api/v1/databases/%s/snapshots/%s/restore", dbID, snapshotID), nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to restore snapshot: %v", err)), nil
	}

	if !resp.Success && resp.Error != "" {
		return mcp.NewToolResultError(fmt.Sprintf("Snapshot restore error: %s", resp.Error)), nil
	}

	resultJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Snapshot restoration initiated:\n%s", string(resultJSON))), nil
}

// Helper functions

// getStringArg gets a string argument with a default value
func getStringArg(request mcp.CallToolRequest, key, defaultValue string) string {
	args := request.Params.Arguments
	argsMap, _ := args.(map[string]interface{})
	if val, ok := argsMap[key]; ok {
		if strVal, ok := val.(string); ok && strVal != "" {
			return strVal
		}
	}
	return defaultValue
}

// truncateSQL truncates SQL queries for logging
func truncateSQL(sql string) string {
	if len(sql) > 200 {
		return sql[:200] + "..."
	}
	return sql
}
