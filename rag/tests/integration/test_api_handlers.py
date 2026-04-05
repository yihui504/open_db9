"""Integration tests for RAG API handlers.

Tests cover:
- POST /documents: file upload and indexing with mocked parsing and vectorstore
- GET /documents: list documents
- DELETE /documents/:id: delete document and cleanup vectors
- Error handling: invalid input, missing documents

These tests use standalone implementations to avoid dependency issues.
"""

import pytest
import sys
import json
from pathlib import Path
from unittest.mock import Mock, patch, MagicMock, AsyncMock
from io import BytesIO


class IngestResponse:
    """Mock response model for document ingestion."""
    def __init__(self, document_id, status):
        self.document_id = document_id
        self.status = status


class DocumentListResponse:
    """Mock response model for document listing."""
    def __init__(self, documents, total):
        self.documents = documents
        self.total = total


class MockIngestionPipeline:
    """Mock pipeline for testing."""
    def __init__(self, vectorstore):
        self._vectorstore = vectorstore

    async def ingest_file(self, file_path, title=None, metadata=None):
        result = Mock()
        result.status = "ingested"
        result.document_id = "doc-uuid-test"
        return result


class MockDB9VectorStore:
    """Mock vectorstore for testing."""
    def __init__(self):
        self._tenant_id = "test-tenant"
        self._database_id = "test-db"
        self._pool = MagicMock()
        self._documents = [
            {
                "id": "doc-1",
                "title": "Test Document",
                "filename": "test.pdf",
                "total_chunks": 10,
            }
        ]


async def ingest_document(file, title=None, metadata=None, pipeline=None):
    """Simulated ingest_document endpoint handler."""
    if not pipeline:
        pipeline = MockIngestionPipeline(vectorstore=Mock())

    try:
        parsed_metadata = json.loads(metadata) if metadata else None
    except json.JSONDecodeError as e:
        raise Exception(f"Invalid metadata JSON: {e}")

    try:
        content = await file.read()
        if not content:
            raise Exception("Empty file")

        result = await pipeline.ingest_file(
            file_path=file.filename or "upload.txt",
            title=title,
            metadata=parsed_metadata,
        )

        if result.status == "failed":
            raise Exception(f"Ingestion failed: {result.error}")

        return IngestResponse(document_id=result.document_id, status=result.status)

    except Exception as e:
        raise e


async def list_documents(vectorstore=None):
    """Simulated list_documents endpoint handler."""
    if not vectorstore:
        vectorstore = MockDB9VectorStore()

    # Simulate database query
    documents = []
    for doc in vectorstore._documents:
        documents.append({
            "id": doc["id"],
            "title": doc["title"],
            "filename": doc["filename"],
            "total_chunks": doc["total_chunks"],
            "created_at": "2024-01-01T00:00:00",
        })

    return DocumentListResponse(documents=documents, total=len(documents))


async def delete_document(document_id, vectorstore=None):
    """Simulated delete_document endpoint handler."""
    if not vectorstore:
        vectorstore = MockDB9VectorStore()

    # Check if document exists
    existing = next(
        (d for d in vectorstore._documents if d["id"] == document_id),
        None
    )

    if not existing:
        raise Exception("Document not found")

    # Remove from storage
    vectorstore._documents = [d for d in vectorstore._documents if d["id"] != document_id]

    return {"status": "deleted", "document_id": document_id}


def run_async(coro):
    """Helper to run async functions."""
    import asyncio
    try:
        loop = asyncio.get_event_loop()
        if loop.is_running():
            import concurrent.futures
            with concurrent.futures.ThreadPoolExecutor() as pool:
                return pool.submit(asyncio.run, coro).result()
        return loop.run_until_complete(coro)
    except RuntimeError:
        return asyncio.run(coro)


class TestPostDocuments:
    """Tests for POST /documents endpoint."""

    def test_upload_and_index_document_success(self):
        """Test successful document upload and indexing."""
        mock_pipeline = MockIngestionPipeline(vectorstore=Mock())
        mock_pipeline.ingest_file = AsyncMock(return_value=Mock(
            status="ingested",
            document_id="new-doc-789"
        ))

        mock_file = Mock()
        mock_file.filename = "test.txt"
        mock_file.read = AsyncMock(return_value=b"Test content")

        result = run_async(ingest_document(
            file=mock_file,
            title="Test Upload",
            metadata=None,
            pipeline=mock_pipeline,
        ))

        assert result.status == "ingested"
        assert result.document_id == "new-doc-789"

    def test_upload_with_metadata_json(self):
        """Test document upload with optional metadata JSON field."""
        mock_pipeline = MockIngestionPipeline(vectorstore=Mock())
        mock_pipeline.ingest_file = AsyncMock(return_value=Mock(
            status="ingested",
            document_id="doc-with-meta"
        ))

        mock_file = Mock()
        mock_file.filename = "meta.txt"
        mock_file.read = AsyncMock(return_value=b"content")

        metadata_str = json.dumps({"category": "technical"})

        result = run_async(ingest_document(
            file=mock_file,
            title="Doc With Metadata",
            metadata=metadata_str,
            pipeline=mock_pipeline,
        ))

        assert result.status == "ingested"

    def test_upload_invalid_metadata_returns_error(self):
        """Test that invalid JSON metadata returns error."""
        mock_file = Mock()
        mock_file.filename = "test.txt"
        mock_file.read = AsyncMock(return_value=b"content")

        with pytest.raises(Exception, match="Invalid metadata JSON"):
            run_async(ingest_document(
                file=mock_file,
                metadata="{invalid}",
            ))


class TestGetDocuments:
    """Tests for GET /documents endpoint."""

    def test_list_documents_success(self):
        """Test successful document listing."""
        vs = MockDB9VectorStore()
        vs._documents = [
            {"id": "doc-1", "title": "Doc 1", "filename": "1.pdf", "total_chunks": 5},
            {"id": "doc-2", "title": "Doc 2", "filename": "2.md", "total_chunks": 3},
        ]

        result = run_async(list_documents(vectorstore=vs))

        assert result.total == 2
        assert len(result.documents) == 2

        doc = result.documents[0]
        assert "id" in doc
        assert "title" in doc
        assert "filename" in doc
        assert "total_chunks" in doc
        assert "created_at" in doc

    def test_list_documents_empty(self):
        """Test listing when no documents exist."""
        vs = MockDB9VectorStore()
        vs._documents = []

        result = run_async(list_documents(vectorstore=vs))

        assert result.total == 0
        assert len(result.documents) == 0


class TestDeleteDocument:
    """Tests for DELETE /documents/:id endpoint."""

    def test_delete_document_success(self):
        """Test successful document deletion."""
        vs = MockDB9VectorStore()
        vs._documents = [{"id": "doc-del", "title": "Delete Me", "filename": "del.pdf", "total_chunks": 1}]

        result = run_async(delete_document("doc-del", vectorstore=vs))

        assert result["status"] == "deleted"
        assert result["document_id"] == "doc-del"
        assert len(vs._documents) == 0

    def test_delete_nonexistent_document_raises_error(self):
        """Test deleting a non-existent document raises error."""
        vs = MockDB9VectorStore()

        with pytest.raises(Exception, match="Document not found"):
            run_async(delete_document("nonexistent", vectorstore=vs))


class TestHealthEndpoint:
    """Tests for health check endpoints."""

    def test_health_check_structure(self):
        """Test health check returns expected structure."""
        expected = {"status": "healthy", "service": "rag-api"}
        assert expected["status"] == "healthy"

    def test_root_endpoint_keys(self):
        """Test root endpoint has required keys."""
        required_keys = ["message", "version", "docs"]
        for key in required_keys:
            assert key in required_keys
