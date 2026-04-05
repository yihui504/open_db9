"""Pydantic models for RAG API request/response validation."""

from pydantic import BaseModel, Field
from typing import List, Optional, Dict, Any


class IngestRequest(BaseModel):
    file_path: str = Field(..., description="Path to file to ingest")
    title: Optional[str] = Field(None, description="Document title")
    metadata: Optional[Dict[str, Any]] = Field(default_factory=dict)


class IngestResponse(BaseModel):
    document_id: str
    status: str


class QueryRequest(BaseModel):
    question: str = Field(..., description="Question to answer")
    k: Optional[int] = Field(4, description="Number of chunks to retrieve")
    filter: Optional[Dict[str, Any]] = Field(None, description="Metadata filter")


class SourceDocument(BaseModel):
    content: str
    metadata: Dict[str, Any]
    similarity: float


class QueryResponse(BaseModel):
    question: str
    answer: str
    sources: List[Dict[str, Any]]
    latency_ms: float


class DocumentMetadata(BaseModel):
    id: str
    title: str
    filename: str
    total_chunks: int
    created_at: str


class DocumentListResponse(BaseModel):
    documents: List[DocumentMetadata]
    total: int
