"""Simulated real-world usage of the RAG system."""
import sys
import asyncio
from pathlib import Path
import psycopg

sys.path.insert(0, "/app/src")
sys.path.insert(0, "/app")

from src.config.settings import RAGSettings
from src.vectorstore.db9_vectorstore import DB9VectorStore
from src.ingestion.parsers import TextParser
from src.ingestion.chunkers import DocumentChunker
from src.query.engine import RAGQueryEngine


async def main():
    print("=" * 70)
    print("🚀 OPEN-DB9 RAG: REAL-WORLD TEST")
    print("=" * 70)

    settings = RAGSettings()
    tenant_id = "00000000-0000-0000-0000-000000000001"
    database_id = "00000000-0000-0000-0000-000000000002"
    conn_string = settings.get_connection_string()

    print(f"\n1. Initializing System Context...")
    try:
        with psycopg.connect(conn_string, autocommit=True) as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "INSERT INTO tenants (id, name, slug) VALUES (%s, %s, %s) ON CONFLICT (id) DO NOTHING",
                    (tenant_id, "Test Tenant", "test-tenant"),
                )
                cur.execute(
                    "INSERT INTO databases (id, tenant_id, name, slug, engine, version, region, size) VALUES (%s, %s, %s, %s, %s, %s, %s, %s) ON CONFLICT (id) DO NOTHING",
                    (database_id, tenant_id, "Test DB", "test-db", "postgresql", "16", "us-east-1", "small"),
                )
        print("   ✓ Tenant and Database context initialized in DB")
    except Exception as e:
        print(f"   ⚠ DB Context init warning: {e}")

    print(f"   Provider: {settings.provider}")
    print(f"   Embedding dimension: {settings.get_embedding_dimension()}")

    vectorstore = DB9VectorStore(
        connection_string=conn_string,
        embedding_function=settings.get_embeddings(),
        tenant_id=tenant_id,
        database_id=database_id,
    )

    print(f"\n2. Ingesting Document: README.md")
    readme_path = Path("/app/README.md")
    parser = TextParser()
    parsed = parser.parse(str(readme_path))
    chunker = DocumentChunker(chunk_size=500, chunk_overlap=50)
    chunks = chunker.chunk(parsed["content"], parsed["metadata"])

    print(f"   Parsed {len(parsed['content'])} chars into {len(chunks)} chunks")

    texts = [c["content"] for c in chunks]
    metadatas = [c["metadata"] for c in chunks]
    doc_id = "00000000-0000-0000-0000-000000000003"

    try:
        with psycopg.connect(conn_string, autocommit=True) as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "INSERT INTO rag_documents (id, tenant_id, database_id, title, filename) VALUES (%s, %s, %s, %s, %s) ON CONFLICT (id) DO NOTHING",
                    (doc_id, tenant_id, database_id, "README", "README.md"),
                )
    except Exception as e:
        print(f"   ⚠ Document record init warning: {e}")

    vectorstore.add_texts(texts=texts, metadatas=metadatas, document_id=doc_id)
    print("   ✓ Chunks stored in pgvector")

    print(f"\n3. Performing Real Query...")
    engine = RAGQueryEngine(vectorstore=vectorstore)
    question = "这个 RAG 系统支持哪些 AI 供应商？它是如何保证数据隔离的？"
    print(f"   Question: {question}")

    print(f"\n   Generating Answer...")
    result = engine.query(question=question)

    print(f"\n{'=' * 10} ANSWER {'=' * 10}")
    print(result["answer"])
    print(f"{'=' * 28}")
    print(f"\n   Sources used: {len(result['sources'])}")
    print(f"   Latency: {result['latency_ms']:.2f}ms")
    print("\n" + "=" * 70)
    print("🏁 TEST COMPLETE")
    print("=" * 70)


if __name__ == "__main__":
    asyncio.run(main())
