"""Simulated real-world usage of the RAG system."""
import sys
import asyncio

sys.path.insert(0, "/app/src")
sys.path.insert(0, "/app")

from src.config.settings import RAGSettings
from src.vectorstore.db9_vectorstore import DB9VectorStore
from src.query.engine import RAGQueryEngine

async def main():
    settings = RAGSettings()

    tenant_id = "00000000-0000-0000-0000-000000000001"
    database_id = "00000000-0000-0000-0000-000000000002"
    conn_string = settings.get_connection_string()

    vectorstore = DB9VectorStore(
        connection_string=conn_string,
        embedding_function=settings.get_embeddings(),
        tenant_id=tenant_id,
        database_id=database_id
    )

    engine = RAGQueryEngine(vectorstore=vectorstore)

    question = "Saga 模式在这里起什么作用？"
    result = engine.query(question=question)

    print(f"QUESTION: {question}")
    print(f"ANSWER: {result['answer']}")

if __name__ == "__main__":
    asyncio.run(main())
