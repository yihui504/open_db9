"""Demo script for RAG system."""

import os
import sys
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from src.config.settings import get_settings
from src.vectorstore.db9_vectorstore import DB9VectorStore
from src.query.engine import RAGQueryEngine


def demo():
    """Run RAG system demo."""
    settings = get_settings()

    print("=== Open-DB9 RAG System Demo ===\n")

    # Create vectorstore
    print("1. Creating VectorStore...")
    vectorstore = DB9VectorStore(
        connection_string=settings.get_connection_string(),
        embedding_function=settings.get_embeddings(),
        tenant_id="demo-tenant-id",
        database_id="demo-database-id",
    )
    print("   ✓ VectorStore created\n")

    # Create query engine
    print("2. Creating Query Engine...")
    engine = RAGQueryEngine(vectorstore=vectorstore)
    print("   ✓ Query Engine created\n")

    # Example query
    print("3. Running example query...")
    print("   Question: What is Open-DB9?")
    print()

    try:
        result = engine.query(
            question="What is Open-DB9?",
            k=4
        )

        print(f"   Answer: {result['answer']}")
        print(f"   Sources: {len(result['sources'])}")
        print(f"   Latency: {result['latency_ms']:.0f}ms")
    except Exception as e:
        print(f"   Note: Query requires ingested documents - {e}")

    print("\n=== Demo Complete ===")
    print("\nNext steps:")
    print("  1. Ingest documents: python scripts/ingest_documents.py ./documents/")
    print("  2. Query the system: python -m rag.src.cli.commands.query 'Your question'")


if __name__ == "__main__":
    demo()
