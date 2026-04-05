"""FastAPI routes for RAG system."""

import json
from fastapi import APIRouter, HTTPException, UploadFile, File, Depends, Form
from typing import Optional, Dict, Any

from ..ingestion.pipeline import IngestionPipeline
from ..query.engine import RAGQueryEngine
from ..vectorstore.db9_vectorstore import DB9VectorStore
from ..config.settings import get_settings
from .auth import get_current_tenant_id
from .models import (
    IngestResponse,
    QueryRequest,
    QueryResponse,
    DocumentListResponse,
)

router = APIRouter(prefix="/databases/{database_id}/rag", tags=["RAG"])

_settings = get_settings()


# Dependency injection
# NOTE: database_id comes from the path parameter: /api/v1/databases/{id}/rag/...
# This is injected by FastAPI's path parameter system
def get_vectorstore(
    database_id: str,  # From path parameter /databases/{id}/rag/...
    tenant_id: str = Depends(get_current_tenant_id)
) -> DB9VectorStore:
    """Get VectorStore instance for the current tenant and database.

    Args:
        database_id: UUID from path parameter /databases/{id}/rag/...
        tenant_id: UUID from JWT token claims

    Returns:
        Configured DB9VectorStore instance
    """
    return DB9VectorStore(
        connection_string=_settings.get_connection_string(),
        embedding_function=_settings.get_embeddings(),
        tenant_id=tenant_id,
        database_id=database_id,  # Now from path parameter, not hardcoded
    )


def get_ingestion_pipeline(
    vectorstore: DB9VectorStore = Depends(get_vectorstore)
) -> IngestionPipeline:
    """Get IngestionPipeline instance."""
    return IngestionPipeline(vectorstore=vectorstore)


def get_query_engine(
    vectorstore: DB9VectorStore = Depends(get_vectorstore)
) -> RAGQueryEngine:
    """Get RAGQueryEngine instance."""
    return RAGQueryEngine(vectorstore=vectorstore)


@router.post("/documents", response_model=IngestResponse)
async def ingest_document(
    file: UploadFile = File(...),
    title: Optional[str] = Form(default=None),
    metadata: Optional[str] = Form(default=None),
    pipeline: IngestionPipeline = Depends(get_ingestion_pipeline),
):
    import tempfile
    import os

    try:
        parsed_metadata = json.loads(metadata) if metadata else None
    except json.JSONDecodeError as e:
        raise HTTPException(status_code=400, detail=f"Invalid metadata JSON: {str(e)}")

    try:
        # Save uploaded file temporarily
        with tempfile.NamedTemporaryFile(delete=False, suffix=file.filename) as tmp:
            content = await file.read()
            tmp.write(content)
            tmp_path = tmp.name

        result = await pipeline.ingest_file(
            file_path=tmp_path,
            title=title,
            metadata=parsed_metadata,
        )

        # Clean up temp file
        os.unlink(tmp_path)

        if result.status == "failed":
            raise HTTPException(
                status_code=500,
                detail=f"Ingestion failed: {result.error}"
            )

        return IngestResponse(document_id=result.document_id, status=result.status)

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/query", response_model=QueryResponse)
async def query_rag(
    request: QueryRequest,
    engine: RAGQueryEngine = Depends(get_query_engine),
):
    """Query the RAG system."""
    try:
        result = engine.query(
            question=request.question,
            k=request.k,
            filter=request.filter,
        )
        return QueryResponse(**result)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/documents", response_model=DocumentListResponse)
async def list_documents(
    vectorstore: DB9VectorStore = Depends(get_vectorstore),
):
    try:
        with vectorstore._pool.connection() as conn:
            conn.execute("SELECT set_rag_tenant_id(%s)", [vectorstore._tenant_id])
            with conn.cursor() as cur:
                cur.execute(
                    """
                    SELECT id, title, filename, total_chunks, created_at
                    FROM rag_documents
                    WHERE database_id = %s
                    ORDER BY created_at DESC
                    """,
                    [vectorstore._database_id],
                )
                rows = cur.fetchall()

        documents = [
            {
                "id": str(row[0]),
                "title": row[1],
                "filename": row[2],
                "total_chunks": row[3],
                "created_at": row[4].isoformat(),
            }
            for row in rows
        ]
        return DocumentListResponse(documents=documents, total=len(documents))
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.delete("/documents/{document_id}")
async def delete_document(
    document_id: str,
    vectorstore: DB9VectorStore = Depends(get_vectorstore),
):
    try:
        with vectorstore._pool.connection() as conn:
            conn.execute("SELECT set_rag_tenant_id(%s)", [vectorstore._tenant_id])
            with conn.cursor() as cur:
                cur.execute(
                    """
                    DELETE FROM rag_documents
                    WHERE id = %s AND database_id = %s
                    RETURNING id
                    """,
                    [document_id, vectorstore._database_id],
                )
                deleted = cur.fetchone()
                conn.commit()

        if deleted is None:
            raise HTTPException(status_code=404, detail="Document not found")

        return {"status": "deleted", "document_id": document_id}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
