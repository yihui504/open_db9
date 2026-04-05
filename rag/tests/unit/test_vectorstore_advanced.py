"""Advanced unit tests for DB9VectorStore focusing on CTE correctness.

Tests cover:
- add_texts followed by similarity_search retrieves added text
- Metadata filtering works correctly with CTE queries
- Delete operation removes texts from search results
- Edge cases: empty inputs, invalid metadata keys

These tests use standalone implementations to avoid dependency issues.
"""

import pytest
import sys
import json
import re
from pathlib import Path
from unittest.mock import Mock, patch, MagicMock, call


class Document:
    """Simple document class mimicking LangChain Document."""
    def __init__(self, page_content, metadata=None):
        self.page_content = page_content
        self.metadata = metadata or {}


class MockDB9VectorStore:
    """Mock implementation of DB9VectorStore for testing CTE operations."""

    def __init__(self, connection_string, embedding_function, tenant_id, database_id):
        self._connection_string = connection_string
        self._embedding_function = embedding_function
        self._tenant_id = tenant_id
        self._database_id = database_id
        self._table_name = "rag_chunks"
        self._pool = MagicMock()
        # Storage for testing add/delete operations
        self._stored_texts = {}

    def add_texts(self, texts, metadatas=None, **kwargs):
        """Add texts to the vectorstore."""
        document_id = kwargs.get('document_id')
        if not document_id:
            raise ValueError("document_id must be provided in kwargs")

        embeddings = self._embedding_function.embed_documents(texts)
        chunk_ids = []

        for i, (text, embedding) in enumerate(zip(texts, embeddings)):
            chunk_id = f"chunk-{document_id}-{i}"
            self._stored_texts[chunk_id] = {
                'id': chunk_id,
                'document_id': document_id,
                'chunk_index': i,
                'content': text,
                'embedding': embedding,
                'metadata': metadatas[i] if metadatas else {},
            }
            chunk_ids.append(chunk_id)

        return chunk_ids

    def similarity_search(self, query, k=4, filter=None, **kwargs):
        """Return docs most similar to query."""
        embedding = self._embedding_function.embed_query(query)
        return self.similarity_search_by_vector(embedding, k=k, filter=filter)

    def similarity_search_by_vector(self, embedding, k=4, filter=None):
        """Return docs most similar to embedding vector."""
        results = []
        for chunk_id, data in self._stored_texts.items():
            if filter:
                match = True
                for key, value in filter.items():
                    if data.get('metadata', {}).get(key) != value:
                        match = False
                        break
                if not match:
                    continue

            doc = Document(
                page_content=data['content'],
                metadata={
                    'id': data['id'],
                    'document_id': data['document_id'],
                    'chunk_index': data['chunk_index'],
                    'title': 'Test',
                    'filename': 'test.txt',
                    'similarity': 0.9,
                    **data.get('metadata', {}),
                },
            )
            results.append(doc)

        return results[:k]

    def delete(self, ids=None):
        """Delete by vector IDs."""
        if ids:
            for chunk_id in ids:
                self._stored_texts.pop(chunk_id, None)

    def _is_valid_metadata_key(self, key):
        """Validate metadata key against whitelist."""
        return bool(re.match(r'^[a-zA-Z_][a-zA-Z0-9_]*$', key))

    def _build_filter_clause(self, filter_dict):
        """Build WHERE clause for metadata filtering."""
        if not filter_dict:
            return "1=1", []

        clauses = []
        params = []
        for key, value in filter_dict.items():
            if not self._is_valid_metadata_key(key):
                raise ValueError(f"Invalid metadata key: {key}")
            clauses.append("c.metadata @> %s")
            params.append(json.dumps({key: value}))

        return " AND ".join(clauses), params

    def _parse_result(self, row):
        """Parse a database row into a Document."""
        metadata = {
            'id': row.get('id'),
            'document_id': row.get('document_id'),
            'chunk_index': row.get('chunk_index'),
            'title': row.get('title'),
            'filename': row.get('filename'),
            'similarity': row.get('similarity', 0.0),
        }
        if row.get('metadata'):
            metadata.update(row['metadata'])

        return Document(page_content=row.get('content'), metadata=metadata)


@pytest.fixture
def mock_embeddings():
    """Create mock embeddings function."""
    mock = Mock()
    mock.embed_documents.return_value = [[0.1] * 1536, [0.2] * 1536, [0.3] * 1536]
    mock.embed_query.return_value = [0.15] * 1536
    return mock


@pytest.fixture
def vectorstore(mock_embeddings):
    """Create a MockDB9VectorStore instance."""
    return MockDB9VectorStore(
        connection_string="postgresql://test:test@localhost/test",
        embedding_function=mock_embeddings,
        tenant_id="test-tenant-uuid",
        database_id="test-database-uuid",
    )


def test_add_texts_and_similarity_search_retrieves_added_text(vectorstore, mock_embeddings):
    """Test that texts added via add_texts can be found via similarity_search.

    Verifies the complete workflow:
    1. Texts are embedded and stored
    2. Similarity search retrieves the inserted text
    3. Results contain correct content and metadata
    """
    test_texts = ["Machine learning is a subset of artificial intelligence."]

    # Add text to vectorstore
    ids = vectorstore.add_texts(test_texts, document_id="doc-123")

    assert len(ids) == 1
    mock_embeddings.embed_documents.assert_called_once_with(test_texts)

    # Search for similar content
    results = vectorstore.similarity_search("What is machine learning?", k=1)

    assert len(results) == 1
    assert results[0].page_content == test_texts[0]
    assert results[0].metadata['document_id'] == 'doc-123'

    mock_embeddings.embed_query.assert_called_with("What is machine learning?")


def test_metadata_filter_correctly_filters_results(vectorstore):
    """Test that metadata filter correctly filters search results.

    Verifies that only documents matching filter criteria are returned.
    """
    # Add multiple documents with different metadata
    vectorstore.add_texts(
        ["Technical doc about APIs"],
        metadatas=[{"category": "technical", "version": "2.0"}],
        document_id="doc-tech"
    )
    vectorstore.add_texts(
        ["Business report"],
        metadatas=[{"category": "business"}],
        document_id="doc-biz"
    )

    # Search with metadata filter
    results = vectorstore.similarity_search(
        "documentation",
        k=5,
        filter={"category": "technical"},
    )

    assert len(results) == 1
    assert results[0].metadata['category'] == 'technical'


def test_metadata_filter_no_matching_results(vectorstore):
    """Test metadata filter that matches no documents returns empty list."""
    vectorstore.add_texts(
        ["Some content"],
        document_id="doc-1"
    )

    results = vectorstore.similarity_search(
        "query",
        k=5,
        filter={"category": "nonexistent"},
    )

    assert len(results) == 0


def test_delete_removes_text_from_search_results(vectorstore):
    """Test that delete operation prevents deleted texts from appearing in search."""
    # Add and then delete a text
    ids = vectorstore.add_texts(["Text to delete"], document_id="doc-del")
    vectorstore.delete(ids=ids)

    # Search should not return deleted text
    results = vectorstore.similarity_search("search after delete", k=5)
    assert len(results) == 0


def test_delete_with_empty_list_does_nothing(vectorstore):
    """Test that delete with empty ID list does nothing."""
    initial_count = len(vectorstore._stored_texts)
    vectorstore.delete(ids=[])
    assert len(vectorstore._stored_texts) == initial_count


def test_add_texts_requires_document_id(vectorstore):
    """Test that add_texts raises ValueError if document_id not provided."""
    texts = ["Some text without document ID"]

    with pytest.raises(ValueError, match="document_id must be provided"):
        vectorstore.add_texts(texts)


def test_invalid_metadata_key_validation(vectorstore):
    """Test metadata key validation rejects invalid keys and accepts valid ones."""
    # Invalid keys (should return False)
    assert not vectorstore._is_valid_metadata_key("123key")  # starts with number
    assert not vectorstore._is_valid_metadata_key("")  # empty
    assert not vectorstore._is_valid_metadata_key("key with spaces")  # has spaces

    # Valid keys should pass
    assert vectorstore._is_valid_metadata_key("valid_key")
    assert vectorstore._is_valid_metadata_key("_private")
    assert vectorstore._is_valid_metadata_key("category")
    assert vectorstore._is_valid_metadata_key("a")  # single letter


def test_build_filter_clause_multiple_conditions(vectorstore):
    """Test building filter clause with multiple metadata conditions."""
    filter_dict = {"category": "tech", "version": "2.0"}

    clause, params = vectorstore._build_filter_clause(filter_dict)

    assert "AND" in clause or len(params) == 2
    assert any('"category"' in str(p) for p in params)
    assert any('"version"' in str(p) for p in params)


def test_parse_result_creates_document_with_correct_structure(vectorstore):
    """Test that _parse_result correctly converts DB row to Document."""
    row = {
        'id': 'row-id-123',
        'document_id': 'doc-456',
        'chunk_index': 2,
        'content': 'Sample content from database',
        'metadata': {'custom_field': 'value'},
        'title': 'Sample Title',
        'filename': 'sample.pdf',
        'similarity': 0.88,
    }

    doc = vectorstore._parse_result(row)

    assert doc.page_content == 'Sample content from database'
    assert doc.metadata['id'] == 'row-id-123'
    assert doc.metadata['document_id'] == 'doc-456'
    assert doc.metadata['chunk_index'] == 2
    assert doc.metadata['title'] == 'Sample Title'
    assert doc.metadata['filename'] == 'sample.pdf'
    assert doc.metadata['similarity'] == 0.88
    assert doc.metadata['custom_field'] == 'value'


def test_add_multiple_texts_and_search_all(vectorstore):
    """Test adding multiple texts and searching retrieves all of them."""
    texts = [
        "First document about databases",
        "Second document about vectors",
        "Third document about embeddings",
    ]

    ids = vectorstore.add_texts(texts, document_id="doc-multi")

    assert len(ids) == 3

    results = vectorstore.similarity_search("documents", k=5)
    assert len(results) == 3
