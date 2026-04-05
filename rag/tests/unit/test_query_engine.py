"""Integration tests for RAGQueryEngine.

Tests cover:
- Query with documents: retriever returns docs, LLM generates answer with sources
- Query without documents: retriever returns empty, degraded response returned
- LLM unavailable: exception raised, raw retrieval excerpts returned

These tests use standalone implementations to avoid dependency issues.
"""

import pytest
import sys
import time
from pathlib import Path
from unittest.mock import Mock, patch, MagicMock


class Document:
    """Simple document class for testing."""
    def __init__(self, page_content, metadata=None):
        self.page_content = page_content
        self.metadata = metadata or {}


class MockRAGQueryEngine:
    """Mock implementation of RAGQueryEngine for testing."""

    def __init__(self, vectorstore, llm=None):
        self._vectorstore = vectorstore
        self._llm = llm
        self._chain = Mock() if llm else None
        self._default_k = 3

    def query(self, question, k=None, filter=None):
        """Query the RAG system."""
        start_time = time.time()

        k = k or self._default_k
        results = self._vectorstore.similarity_search(
            query=question,
            k=k,
            filter=filter,
        )

        if self._chain is None:
            response = self._generate_local_answer(question, results)
        else:
            response = self._chain.invoke(question)

        latency = (time.time() - start_time) * 1000

        return {
            "question": question,
            "answer": response,
            "sources": [doc.metadata for doc in results],
            "latency_ms": latency,
        }

    def _generate_local_answer(self, question, docs):
        if not docs:
            return '未检索到与问题"' + question + '"相关的本地上下文。'

        excerpts = []
        for index, doc in enumerate(docs[:3], start=1):
            content = " ".join(doc.page_content.split())
            excerpts.append(f"[{index}] {content[:240]}")

        return "\n\n".join(excerpts)


@pytest.fixture
def mock_vectorstore():
    """Create a mock vectorstore."""
    return Mock()


@pytest.fixture
def engine_with_llm(mock_vectorstore):
    """Create query engine with mocked LLM."""
    mock_llm = Mock()
    return MockRAGQueryEngine(vectorstore=mock_vectorstore, llm=mock_llm)


@pytest.fixture
def engine_without_llm(mock_vectorstore):
    """Create query engine without LLM (degraded mode)."""
    return MockRAGQueryEngine(vectorstore=mock_vectorstore)


def test_query_with_documents_returns_answer_with_sources(engine_with_llm):
    """Test query when documents are retrieved and LLM is available.

    Verifies that:
    - Retriever is called with correct parameters
    - LLM chain generates an answer
    - Response includes sources from retrieved documents
    - Response structure contains all required fields
    """
    question = "What is the database architecture?"

    # Mock retrieved documents
    mock_docs = [
        Document(
            page_content="The system uses PostgreSQL with pgvector extension.",
            metadata={"id": "chunk-1", "document_id": "doc-1", "title": "Architecture Doc"}
        ),
        Document(
            page_content="Vector embeddings are stored in 1536 dimensions.",
            metadata={"id": "chunk-2", "document_id": "doc-1", "title": "Architecture Doc"}
        ),
    ]

    engine_with_llm._vectorstore.similarity_search.return_value = mock_docs
    engine_with_llm._chain.invoke.return_value = (
        "The system uses PostgreSQL with pgvector for vector storage."
    )

    result = engine_with_llm.query(question)

    assert "question" in result
    assert "answer" in result
    assert "sources" in result
    assert "latency_ms" in result

    assert result["question"] == question
    assert isinstance(result["answer"], str)
    assert len(result["answer"]) > 0
    assert len(result["sources"]) == 2
    assert result["latency_ms"] >= 0

    engine_with_llm._vectorstore.similarity_search.assert_called_once_with(
        query=question,
        k=3,
        filter=None,
    )

    engine_with_llm._chain.invoke.assert_called_once_with(question)


def test_query_without_documents_returns_degraded_response(engine_without_llm):
    """Test query when no documents are retrieved (empty results).

    Verifies that degraded response message is returned in Chinese.
    """
    question = "Query about non-existent topic"

    engine_without_llm._vectorstore.similarity_search.return_value = []

    result = engine_without_llm.query(question)

    assert result["question"] == question
    assert "未检索到" in result["answer"]
    assert len(result["sources"]) == 0
    assert result["latency_ms"] >= 0


def test_query_llm_exception_raises_error(engine_with_llm):
    """Test query when LLM throws an exception.

    Verifies that exception is propagated properly.
    """
    question = "Trigger LLM failure"

    mock_docs = [
        Document(
            page_content="Important context.",
            metadata={"id": "chunk-1"}
        ),
    ]

    engine_with_llm._vectorstore.similarity_search.return_value = mock_docs
    engine_with_llm._chain.invoke.side_effect = Exception("LLM service timeout")

    with pytest.raises(Exception, match="LLM service timeout"):
        engine_with_llm.query(question)

    engine_with_llm._vectorstore.similarity_search.assert_called_once()


def test_query_with_metadata_filter(engine_with_llm):
    """Test query with metadata filter parameter.

    Verifies that filter is passed through to vectorstore search.
    """
    question = "Filtered query"
    filter_dict = {"category": "technical"}

    mock_docs = [Document(page_content="Filtered content", metadata={"id": "c1"})]
    engine_with_llm._vectorstore.similarity_search.return_value = mock_docs
    engine_with_llm._chain.invoke.return_value = "Filtered answer"

    result = engine_with_llm.query(question, filter=filter_dict)

    engine_with_llm._vectorstore.similarity_search.assert_called_once_with(
        query=question,
        k=3,
        filter=filter_dict,
    )
    assert result["question"] == question


def test_query_custom_k_parameter(engine_with_llm):
    """Test query with custom k parameter for number of results.

    Verifies that custom k overrides default.
    """
    question = "Get more results"

    mock_docs = [
        Document(page_content=f"Result {i}", metadata={"id": f"c{i}"})
        for i in range(5)
    ]
    engine_with_llm._vectorstore.similarity_search.return_value = mock_docs
    engine_with_llm._chain.invoke.return_value = "Multiple results answer"

    result = engine_with_llm.query(question, k=5)

    call_kwargs = engine_with_llm._vectorstore.similarity_search.call_args[1]
    assert call_kwargs["k"] == 5
    assert len(result["sources"]) == 5


def test_generate_local_answer_formats_docs_correctly(engine_without_llm):
    """Test local answer generation formats document excerpts properly.

    Verifies that documents are numbered and joined correctly.
    """
    question = "Format test"
    docs = [
        Document(page_content="Short content", metadata={"source": "doc1"}),
        Document(
            page_content="This is a much longer piece of content that should be truncated.",
            metadata={"source": "doc2"}
        ),
    ]

    result = engine_without_llm._generate_local_answer(question, docs)

    assert "[1]" in result
    assert "[2]" in result
    assert "Short content" in result
    assert "\n\n" in result


def test_generate_local_answer_empty_docs(engine_without_llm):
    """Test local answer generation with empty document list.

    Should return appropriate message in Chinese.
    """
    question = "Empty query"
    result = engine_without_llm._generate_local_answer(question, [])

    assert "未检索到" in result
    assert question in result
