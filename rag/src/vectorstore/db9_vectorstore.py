"""Custom LangChain VectorStore for Open-DB9 with pgvector."""

from typing import List, Dict, Any, Optional, Tuple, TypeVar, Type
from langchain_core.vectorstores import VectorStore
from langchain_core.embeddings import Embeddings
from langchain_core.documents import Document
from psycopg import sql
from psycopg_pool import ConnectionPool
from psycopg.rows import dict_row
import numpy as np
import json
import re

V = TypeVar('V', bound='DB9VectorStore')


class DB9VectorStore(VectorStore):
    """Custom LangChain VectorStore for Open-DB9 with pgvector.

    Implements all required LangChain VectorStore interface methods:
    - similarity_search
    - similarity_search_with_score
    - similarity_search_by_vector
    - max_marginal_relevance_search
    - add_texts
    - from_texts
    - add_documents
    """

    def __init__(
        self,
        connection_string: str,
        embedding_function: Embeddings,
        tenant_id: str,
        database_id: str,
        table_name: str = "rag_chunks",
        embedding_dimension: int = 1536,
    ):
        self._connection_string = connection_string
        self._embedding_function = embedding_function
        self._tenant_id = tenant_id
        self._database_id = database_id
        self._table_name = table_name
        self._embedding_dimension = embedding_dimension
        self._pool = ConnectionPool(connection_string)

    @property
    def embeddings(self) -> Embeddings:
        return self._embedding_function

    def add_texts(
        self,
        texts: List[str],
        metadatas: Optional[List[Dict[str, Any]]] = None,
        **kwargs: Any,
    ) -> List[str]:
        """Add texts to the vectorstore.

        Returns:
            List of IDs of the added texts.
        """
        document_id = kwargs.get('document_id')
        if not document_id:
            raise ValueError("document_id must be provided in kwargs")

        # Generate embeddings
        embeddings = self._embedding_function.embed_documents(texts)

        # Insert into database
        chunk_ids = []
        with self._pool.connection() as conn:
            # Set tenant context for RLS
            conn.execute("SELECT set_rag_tenant_id(%s)", [self._tenant_id])

            with conn.cursor() as cur:
                for i, (text, embedding) in enumerate(zip(texts, embeddings)):
                    metadata = metadatas[i] if metadatas else {}
                    chunk_id = self._insert_chunk(
                        cur, document_id, i, text, embedding, metadata
                    )
                    chunk_ids.append(chunk_id)
                conn.commit()

        return chunk_ids

    def _insert_chunk(
        self,
        cursor,
        document_id: str,
        chunk_index: int,
        content: str,
        embedding: List[float],
        metadata: Dict[str, Any],
    ) -> str:
        """Insert a single chunk into the database."""
        query = sql.SQL("""
            INSERT INTO {table} (tenant_id, document_id, chunk_index, content, embedding, metadata)
            VALUES (%s, %s, %s, %s, %s::vector, %s)
            RETURNING id
        """).format(table=sql.Identifier(self._table_name))

        cursor.execute(
            query,
            (self._tenant_id, document_id, chunk_index, content, str(embedding), json.dumps(metadata))
        )
        return cursor.fetchone()[0]

    def similarity_search(
        self,
        query: str,
        k: int = 4,
        filter: Optional[Dict[str, Any]] = None,
        **kwargs: Any,
    ) -> List[Document]:
        """Return docs most similar to query.

        Args:
            query: Input text to search for.
            k: Number of documents to return.
            filter: Filter on metadata fields.

        Returns:
            List of Documents most similar to the query.
        """
        embedding = self._embedding_function.embed_query(query)
        return self.similarity_search_by_vector(embedding, k=k, filter=filter)

    def similarity_search_by_vector(
        self,
        embedding: List[float],
        k: int = 4,
        filter: Optional[Dict[str, Any]] = None,
    ) -> List[Document]:
        """Return docs most similar to embedding vector.

        Args:
            embedding: Embedding vector to search with.
            k: Number of documents to return.
            filter: Filter on metadata fields.

        Returns:
            List of Documents most similar to the embedding.
        """
        with self._pool.connection() as conn:
            # Set tenant context for RLS
            conn.execute("SELECT set_rag_tenant_id(%s)", [self._tenant_id])

            with conn.cursor(row_factory=dict_row) as cur:
                query, params = self._build_search_query(k, filter)
                # Prepend embedding vector to params (it's the first %s in CTE)
                params = [str(embedding)] + params

                cur.execute(query, params)
                results = cur.fetchall()

        return [self._parse_result(row) for row in results]

    def similarity_search_with_score(
        self,
        query: str,
        k: int = 4,
        filter: Optional[Dict[str, Any]] = None,
        **kwargs: Any,
    ) -> List[Tuple[Document, float]]:
        """Return docs and similarity scores most similar to query.

        Args:
            query: Input text to search for.
            k: Number of documents to return.
            filter: Filter on metadata fields.

        Returns:
            List of Tuples of (Document, similarity score).
        """
        embedding = self._embedding_function.embed_query(query)

        with self._pool.connection() as conn:
            # Set tenant context for RLS
            conn.execute("SELECT set_rag_tenant_id(%s)", [self._tenant_id])

            with conn.cursor(row_factory=dict_row) as cur:
                query_sql, params = self._build_search_query(k, filter)
                # Prepend embedding vector to params (it's the first %s in CTE)
                params = [str(embedding)] + params

                cur.execute(query_sql, params)
                results = cur.fetchall()

        return [(self._parse_result(row), row['similarity']) for row in results]

    def _build_search_query(
        self,
        k: int,
        filter: Optional[Dict[str, Any]]
    ) -> Tuple[sql.SQL, List[Any]]:
        """Build the similarity search query using CTE approach.

        Uses a CTE (Common Table Expression) to compute similarity once,
        avoiding the duplicate parameter bug.

        Returns:
            Tuple of (query SQL, parameters list).
        """
        params = []

        # Build filter clause first (if any)
        filter_condition = sql.SQL("1=1")
        if filter:
            filter_condition, filter_params = self._build_filter_clause(filter)
            params.extend(filter_params)

        # CTE approach: compute similarity once in the CTE
        # Only ONE embedding parameter needed (in the CTE, not in ORDER BY)
        base_query = sql.SQL("""
            WITH similarities AS (
                SELECT
                    c.id, c.tenant_id, c.document_id, c.chunk_index, c.content, c.metadata,
                    d.title, d.filename,
                    1 - (c.embedding <=> %s) as similarity
                FROM {table} c
                JOIN rag_documents d ON c.document_id = d.id
                WHERE d.database_id = %s AND {filter}
            )
            SELECT * FROM similarities
            ORDER BY similarity DESC
            LIMIT %s
        """).format(
            table=sql.Identifier(self._table_name),
            filter=filter_condition
        )

        # Parameters: [embedding_vector, database_id, k]
        # Note: filter params already added above if filter exists
        params = [str(self._database_id)] + params
        params.append(k)

        return base_query, params

    def _build_filter_clause(
        self,
        filter_dict: Dict[str, Any]
    ) -> Tuple[sql.SQL, List[Any]]:
        """Build WHERE clause for metadata filtering.

        Uses PostgreSQL @> containment operator for safe JSONB filtering.
        All metadata keys are validated against a whitelist pattern to prevent injection.

        Returns:
            Tuple of (filter clause SQL, parameters list).
        """
        if not filter_dict:
            return sql.SQL("1=1"), []

        clauses = []
        params = []

        for key, value in filter_dict.items():
            # Validate key against whitelist to prevent injection
            if not self._is_valid_metadata_key(key):
                raise ValueError(f"Invalid metadata key: {key}")

            # Use jsonb @> operator for containment check
            # This is safe: we build a JSONB object and check if metadata contains it
            clauses.append(sql.SQL("c.metadata @> %s"))
            params.append(json.dumps({key: value}))

        filter_clause = sql.SQL(" AND ").join(clauses)
        return filter_clause, params

    def _is_valid_metadata_key(self, key: str) -> bool:
        """Validate metadata key against whitelist.

        Only allows alphanumeric characters and underscores,
        starting with a letter or underscore.

        Returns:
            True if key is valid, False otherwise.
        """
        return bool(re.match(r'^[a-zA-Z_][a-zA-Z0-9_]*$', key))

    def _parse_result(self, row: Dict[str, Any]) -> Document:
        """Parse a database row into a LangChain Document."""
        metadata = {
            'id': row['id'],
            'document_id': row['document_id'],
            'chunk_index': row['chunk_index'],
            'title': row['title'],
            'filename': row['filename'],
            'similarity': row.get('similarity', 0.0),
        }
        # Add any additional metadata
        if row.get('metadata'):
            metadata.update(row['metadata'])

        return Document(page_content=row['content'], metadata=metadata)

    def max_marginal_relevance_search(
        self,
        query: str,
        k: int = 4,
        fetch_k: int = 20,
        lambda_mult: float = 0.5,
        filter: Optional[Dict[str, Any]] = None,
        **kwargs: Any,
    ) -> List[Document]:
        """Return docs selected using maximal marginal relevance.

        Maximal marginal relevance optimizes for similarity to query
        AND diversity among selected documents.

        Args:
            query: Input text to search for.
            k: Number of documents to return.
            fetch_k: Number of documents to fetch for MMR.
            lambda_mult: Balance between relevance and diversity.
                        1.0 = only relevance, 0.0 = only diversity.
            filter: Filter on metadata fields.

        Returns:
            List of Documents selected using MMR.
        """
        # Fetch more documents than needed for diversity
        embedding = self._embedding_function.embed_query(query)

        with self._pool.connection() as conn:
            # Set tenant context for RLS
            conn.execute("SELECT set_rag_tenant_id(%s)", [self._tenant_id])

            with conn.cursor(row_factory=dict_row) as cur:
                query_sql, params = self._build_search_query(fetch_k, filter)
                # Prepend embedding vector to params (it's the first %s in CTE)
                params = [str(embedding)] + params

                cur.execute(query_sql, params)
                results = cur.fetchall()

        if not results:
            return []

        # Apply MMR algorithm
        documents = [self._parse_result(row) for row in results]
        embeddings = np.array([self._embedding_function.embed_query(doc.page_content)
                              for doc in documents])

        # Use numpy for efficient MMR computation
        query_embedding = np.array(embedding)
        selected_indices = []
        remaining_indices = list(range(len(documents)))

        for _ in range(min(k, len(documents))):
            if not selected_indices:
                # Select most similar document first
                selected_indices.append(remaining_indices[0])
            else:
                # Compute MMR scores
                selected_embeddings = embeddings[selected_indices]
                max_sim_to_selected = np.max(
                    np.dot(embeddings[remaining_indices], selected_embeddings.T),
                    axis=1
                )

                # MMR = lambda * similarity - (1 - lambda) * max_similarity_to_selected
                query_sim = np.dot(embeddings[remaining_indices], query_embedding)
                mmr_scores = (lambda_mult * query_sim -
                             (1 - lambda_mult) * max_sim_to_selected)

                # Select document with highest MMR score
                best_idx = np.argmax(mmr_scores)
                selected_indices.append(remaining_indices[best_idx])

            # Remove selected from remaining
            remaining_indices.remove(selected_indices[-1])

        return [documents[i] for i in selected_indices]

    @classmethod
    def from_texts(
        cls: Type[V],
        texts: List[str],
        embedding: Embeddings,
        metadatas: Optional[List[Dict[str, Any]]] = None,
        **kwargs: Any,
    ) -> V:
        """Create a VectorStore from a list of texts.

        Args:
            texts: List of texts to add to the vectorstore.
            embedding: Embedding function to use.
            metadatas: Optional list of metadata for each text.
            **kwargs: Additional parameters including:
                - connection_string: Database connection string
                - tenant_id: Tenant UUID
                - database_id: Database UUID
                - document_id: Document UUID

        Returns:
            VectorStore instance with the texts added.
        """
        connection_string = kwargs.get('connection_string')
        tenant_id = kwargs.get('tenant_id')
        database_id = kwargs.get('database_id')

        if not all([connection_string, tenant_id, database_id]):
            raise ValueError("connection_string, tenant_id, and database_id are required")

        vectorstore = cls(
            connection_string=connection_string,
            embedding_function=embedding,
            tenant_id=tenant_id,
            database_id=database_id,
            **{k: v for k, v in kwargs.items()
               if k not in ['connection_string', 'tenant_id', 'database_id']}
        )

        # Add the texts
        vectorstore.add_texts(texts, metadatas=metadatas, **kwargs)

        return vectorstore

    def add_documents(
        self,
        documents: List[Document],
        **kwargs: Any,
    ) -> List[str]:
        """Add documents to the vectorstore.

        Args:
            documents: List of Documents to add.
            **kwargs: Additional parameters.

        Returns:
            List of IDs of the added documents.
        """
        texts = [doc.page_content for doc in documents]
        metadatas = [doc.metadata for doc in documents]
        return self.add_texts(texts, metadatas=metadatas, **kwargs)

    def delete(self, ids: Optional[List[str]] = None) -> None:
        """Delete by vector IDs.

        Args:
            ids: List of IDs to delete.
        """
        if not ids:
            return

        with self._pool.connection() as conn:
            # Set tenant context for RLS
            conn.execute("SELECT set_rag_tenant_id(%s)", [self._tenant_id])

            with conn.cursor() as cur:
                query = sql.SQL("DELETE FROM {table} WHERE id = ANY(%s)").format(
                    table=sql.Identifier(self._table_name)
                )
                cur.execute(query, [ids])
                conn.commit()

    def as_retriever(self, **kwargs: Any):
        """Return a retriever interface for this vectorstore.

        Args:
            **kwargs: Keyword arguments to pass to the retriever.

        Returns:
            A VectorStoreRetriever instance.
        """
        from langchain_core.vectorstores import VectorStoreRetriever
        return VectorStoreRetriever(vectorstore=self, **kwargs)

    def __del__(self):
        """Clean up connection pool on deletion."""
        if hasattr(self, '_pool'):
            self._pool.close()
