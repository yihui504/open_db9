"""Unit tests for DB9VectorStore."""

import pytest
from unittest.mock import Mock, patch, MagicMock
import sys
from pathlib import Path

# Add src to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "src"))

from vectorstore.db9_vectorstore import DB9VectorStore


@pytest.fixture
def mock_embeddings():
    mock = Mock()
    mock.embed_documents.return_value = [[0.1] * 1536, [0.2] * 1536]
    mock.embed_query.return_value = [0.1] * 1536
    return mock


@pytest.fixture
def vectorstore(mock_embeddings):
    return DB9VectorStore(
        connection_string="postgresql://test:test@localhost/test",
        embedding_function=mock_embeddings,
        tenant_id="test-tenant-uuid",
        database_id="test-database-uuid",
    )


def test_add_texts(vectorstore, mock_embeddings):
    """Test adding texts to vectorstore."""
    texts = ["Test document 1", "Test document 2"]

    with patch.object(vectorstore, '_pool') as mock_pool:
        mock_conn = MagicMock()
        mock_cursor = MagicMock()
        mock_conn.execute.return_value = None
        mock_conn.cursor.return_value.__enter__ = Mock(return_value=mock_cursor)
        mock_conn.cursor.return_value.__exit__ = Mock(return_value=None)
        mock_pool.connection.return_value.__enter__ = Mock(return_value=mock_conn)
        mock_pool.connection.return_value.__exit__ = Mock(return_value=None)
        mock_cursor.fetchone.return_value = ["chunk-id-1"]

        ids = vectorstore.add_texts(texts, document_id="doc-id")

        assert len(ids) == 2
        mock_embeddings.embed_documents.assert_called_once()


def test_similarity_search(vectorstore, mock_embeddings):
    """Test similarity search."""
    with patch.object(vectorstore, '_pool') as mock_pool:
        mock_conn = MagicMock()
        mock_cursor = MagicMock()
        mock_conn.execute.return_value = None
        mock_conn.cursor.return_value.__enter__ = Mock(return_value=mock_cursor)
        mock_conn.cursor.return_value.__exit__ = Mock(return_value=None)
        mock_pool.connection.return_value.__enter__ = Mock(return_value=mock_conn)
        mock_pool.connection.return_value.__exit__ = Mock(return_value=None)

        # Mock fetchall to return dict-like rows
        mock_cursor.fetchall.return_value = [
            {
                'id': 'id1',
                'tenant_id': 'tenant1',
                'document_id': 'doc1',
                'chunk_index': 0,
                'content': 'content 1',
                'metadata': {},
                'title': 'title',
                'filename': 'file',
                'similarity': 0.9,
            },
        ]

        results = vectorstore.similarity_search("test query", k=1)

        assert len(results) == 1
        mock_embeddings.embed_query.assert_called_once_with("test query")


def test_max_marginal_relevance_search(vectorstore, mock_embeddings):
    """Test MMR search."""
    with patch.object(vectorstore, '_pool') as mock_pool:
        mock_conn = MagicMock()
        mock_cursor = MagicMock()
        mock_conn.execute.return_value = None
        mock_conn.cursor.return_value.__enter__ = Mock(return_value=mock_cursor)
        mock_conn.cursor.return_value.__exit__ = Mock(return_value=None)
        mock_pool.connection.return_value.__enter__ = Mock(return_value=mock_conn)
        mock_pool.connection.return_value.__exit__ = Mock(return_value=None)

        # Return multiple results for diversity
        mock_cursor.fetchall.return_value = [
            {
                'id': f'id{i}',
                'tenant_id': 'tenant1',
                'document_id': 'doc1',
                'chunk_index': i,
                'content': f'content {i}',
                'metadata': {},
                'title': 'title',
                'filename': 'file',
                'similarity': 0.9 - (i * 0.1),
            }
            for i in range(5)
        ]

        results = vectorstore.max_marginal_relevance_search(
            "test query", k=3, fetch_k=5, lambda_mult=0.5
        )

        assert len(results) == 3
