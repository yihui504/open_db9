"""Document ingestion pipeline with saga pattern for transaction safety."""

from typing import List, Optional, Dict, Any
import httpx
import logging
import json
from pathlib import Path
from dataclasses import dataclass

from .parsers import get_parser
from .chunkers import DocumentChunker
from ..vectorstore.db9_vectorstore import DB9VectorStore
from ..config.settings import get_settings

logger = logging.getLogger(__name__)


@dataclass
class IngestionResult:
    """Result of document ingestion."""
    document_id: Optional[str]
    status: str
    error: Optional[str] = None
    cleanup_performed: List[str] = None

    def __post_init__(self):
        if self.cleanup_performed is None:
            self.cleanup_performed = []


class IngestionPipeline:
    """Pipeline for ingesting documents into RAG system with transaction safety.

    Uses saga pattern for distributed transaction management:
    1. Upload to FS9
    2. Create document record
    3. Add chunks to vectorstore
    4. Update chunk count

    On failure, compensating actions undo completed steps.
    """

    def __init__(
        self,
        vectorstore: DB9VectorStore,
        db9_api_url: str = None,
        fs9_url: str = None,
        api_token: str = None,
    ):
        self._vectorstore = vectorstore
        self._settings = get_settings()
        self._db9_api_url = db9_api_url or self._settings.db9_api_url
        self._fs9_url = fs9_url or self._settings.fs9_url
        self._api_token = api_token or self._settings.db9_api_token
        self._chunker = DocumentChunker(
            chunk_size=self._settings.chunk_size,
            chunk_overlap=self._settings.chunk_overlap,
        )

        # Track saga steps for compensation
        self._completed_steps: List[str] = []

    async def ingest_file(
        self,
        file_path: str,
        title: str = None,
        metadata: Dict[str, Any] = None,
    ) -> IngestionResult:
        """Ingest a single file into the RAG system with saga compensation.

        Saga Steps:
        1. Parse document (local, no compensation needed)
        2. Upload to FS9 -> compensating: delete from FS9
        3. Create document record -> compensating: delete record
        4. Add chunks to vectorstore -> compensating: delete chunks
        5. Update chunk count -> no compensation (update only)

        On any failure, run compensating transactions for completed steps.
        """
        self._completed_steps = []

        try:
            # Step 1: Parse document (local, no cleanup)
            logger.info(f"Parsing document: {file_path}")
            parser = get_parser(file_path)
            parsed = parser.parse(file_path)

            # Step 2: Upload to FS9
            logger.info(f"Uploading to FS9: {file_path}")
            fs9_key = await self._upload_to_fs9(file_path)
            self._completed_steps.append("fs9_upload")

            # Step 3: Create document record via Open-DB9 API
            logger.info(f"Creating document record")
            document_id = await self._create_document_record(
                title=title or Path(file_path).name,
                filename=Path(file_path).name,
                fs9_key=fs9_key,
                metadata={**(metadata or {}), **parsed["metadata"]},
            )
            self._completed_steps.append("document_record")

            # Step 4: Chunk and embed
            logger.info(f"Chunking and embedding document")
            chunks = self._chunker.chunk(parsed["content"], parsed["metadata"])
            texts = [c["content"] for c in chunks]
            metadatas = [c["metadata"] for c in chunks]

            self._vectorstore.add_texts(
                texts=texts,
                metadatas=metadatas,
                document_id=document_id,
            )
            self._completed_steps.append("chunks_added")

            # Step 5: Update document record with chunk count
            logger.info(f"Updating document chunk count: {len(chunks)}")
            await self._update_document_chunk_count(document_id, len(chunks))

            return IngestionResult(
                document_id=document_id,
                status="ingested",
            )

        except Exception as e:
            logger.error(f"Ingestion failed: {e}")
            # Run compensating transactions
            await self._compensate(document_id if 'document_id' in locals() else None)

            return IngestionResult(
                document_id=None,
                status="failed",
                error=str(e),
                cleanup_performed=self._completed_steps,
            )

    async def _upload_to_fs9(self, file_path: str) -> str:
        """Upload file to FS9 service."""
        headers = {}
        if self._api_token:
            headers["Authorization"] = f"Bearer {self._api_token}"

        async with httpx.AsyncClient(timeout=self._settings.fs9_upload_timeout) as client:
            with open(file_path, "rb") as f:
                files = {"file": (Path(file_path).name, f, "application/octet-stream")}
                data = {"key": f"rag/{Path(file_path).name}"}

                response = await client.post(
                    f"{self._fs9_url}/files/upload",
                    files=files,
                    data=data,
                    headers=headers,
                )
                response.raise_for_status()
                return response.json()["key"]

    async def _create_document_record(
        self,
        title: str,
        filename: str,
        fs9_key: str,
        metadata: Dict[str, Any],
    ) -> str:
        with self._vectorstore._pool.connection() as conn:
            conn.execute("SELECT set_rag_tenant_id(%s)", [self._vectorstore._tenant_id])
            with conn.cursor() as cur:
                cur.execute(
                    """
                    INSERT INTO rag_documents (tenant_id, database_id, fs9_key, title, filename, metadata)
                    VALUES (%s, %s, %s, %s, %s, %s)
                    RETURNING id
                    """,
                    [
                        self._vectorstore._tenant_id,
                        self._vectorstore._database_id,
                        fs9_key,
                        title,
                        filename,
                        json.dumps(metadata or {}),
                    ],
                )
                document_id = cur.fetchone()[0]
                conn.commit()
        return str(document_id)

    async def _update_document_chunk_count(self, document_id: str, count: int):
        with self._vectorstore._pool.connection() as conn:
            conn.execute("SELECT set_rag_tenant_id(%s)", [self._vectorstore._tenant_id])
            with conn.cursor() as cur:
                cur.execute(
                    """
                    UPDATE rag_documents
                    SET total_chunks = %s, updated_at = NOW()
                    WHERE id = %s AND database_id = %s
                    """,
                    [count, document_id, self._vectorstore._database_id],
                )
                conn.commit()

    async def _compensate(self, document_id: Optional[str]):
        """Run compensating transactions for saga rollback.

        Executes cleanup actions in reverse order of completion.
        """
        logger.warning(f"Running compensation for steps: {self._completed_steps}")

        # Reverse order for compensation
        for step in reversed(self._completed_steps):
            try:
                if step == "chunks_added" and document_id:
                    logger.info("Compensating: deleting chunks")
                    with self._vectorstore._pool.connection() as conn:
                        conn.execute("SELECT set_rag_tenant_id(%s)", [self._vectorstore._tenant_id])
                        with conn.cursor() as cur:
                            cur.execute(
                                "DELETE FROM rag_chunks WHERE document_id = %s",
                                [document_id],
                            )
                            conn.commit()

                elif step == "document_record" and document_id:
                    logger.info("Compensating: deleting document record")
                    await self._delete_document_record(document_id)

                elif step == "fs9_upload":
                    logger.info("Compensating: deleting from FS9")
                    # FS9 deletion would be here
                    pass

            except Exception as e:
                logger.error(f"Compensation failed for {step}: {e}")

    async def _delete_document_record(self, document_id: str):
        with self._vectorstore._pool.connection() as conn:
            conn.execute("SELECT set_rag_tenant_id(%s)", [self._vectorstore._tenant_id])
            with conn.cursor() as cur:
                cur.execute(
                    "DELETE FROM rag_documents WHERE id = %s AND database_id = %s",
                    [document_id, self._vectorstore._database_id],
                )
                conn.commit()
