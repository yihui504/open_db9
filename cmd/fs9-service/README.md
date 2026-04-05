# FS9 File Storage Service

A RESTful file storage service built with Go's standard library.

## Features

- **RESTful API** for file operations
- **Configurable** via environment variables
- **Health check** endpoint
- **File upload/download/delete** operations
- **File listing** with prefix filtering
- **Content type detection**
- **File size limits**

## Installation

```bash
go build -o fs9-service ./cmd/fs9-service
```

## Configuration

The service is configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `FS9_SERVICE_PORT` | Server port | `9090` |
| `STORAGE_PATH` | Base storage directory | `/data/fs9` |
| `MAX_FILE_SIZE` | Maximum upload size (bytes) | `104857600` (100MB) |

## Running

```bash
# Using defaults
./fs9-service

# With custom configuration
export FS9_SERVICE_PORT=8080
export STORAGE_PATH=/var/data/fs9
export MAX_FILE_SIZE=52428800  # 50MB
./fs9-service
```

## API Endpoints

### Health Check
```
GET /health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2026-03-31T00:24:28Z",
  "service": "fs9"
}
```

### Upload File
```
POST /files/upload
Content-Type: multipart/form-data
```

Parameters:
- `file`: The file to upload (required)
- `key`: Storage key path (optional, defaults to filename)

Response:
```json
{
  "key": "test/file.txt",
  "size": 1024,
  "success": true
}
```

### Download File
```
GET /files/{key}
```

Returns the file content with appropriate headers.

### Delete File
```
DELETE /files/{key}
```

Response:
```json
{
  "key": "test/file.txt",
  "success": true
}
```

### List Files
```
GET /files?prefix={prefix}
```

Parameters:
- `prefix`: Filter files by prefix (optional)

Response:
```json
{
  "count": 1,
  "files": [
    {
      "Key": "test/file.txt",
      "Size": 1024,
      "ModTime": "2026-03-31T00:24:29Z",
      "ContentType": "text/plain"
    }
  ]
}
```

## Usage Examples

### Upload a file
```bash
curl -X POST \
  -F "file=@/path/to/file.txt" \
  -F "key=documents/file.txt" \
  http://localhost:9090/files/upload
```

### Download a file
```bash
curl -O http://localhost:9090/files/documents/file.txt
```

### List all files
```bash
curl http://localhost:9090/files/
```

### List files with prefix
```bash
curl "http://localhost:9090/files/?prefix=documents"
```

### Delete a file
```bash
curl -X DELETE http://localhost:9090/files/documents/file.txt
```

## Architecture

The service uses the `filesystem.Storage` interface, allowing for different storage backends:

- `DiskStorage`: Default implementation using local filesystem
- Pluggable interface for S3, GCS, Azure Blob, etc.

## Logging

The service uses the `logger` package and outputs:
- INFO: Normal operations
- ERROR: Error conditions
- DEBUG: Detailed debugging information

All requests are logged with method, path, and duration.
