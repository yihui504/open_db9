"""Database schema setup script for RAG system."""

import os
import sys
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from src.config.settings import get_settings
import psycopg


def setup_database():
    """Initialize the RAG database schema."""
    settings = get_settings()

    # Read migration file
    # Try multiple paths to be compatible with both local and docker environments
    possible_paths = [
        Path(__file__).parent.parent / "migrations" / "control" / "008_rag_tables.up.sql",
        Path(__file__).parent.parent.parent / "migrations" / "control" / "008_rag_tables.up.sql",
        Path("/app/migrations/control/008_rag_tables.up.sql"),
    ]

    migration_file = None
    for p in possible_paths:
        if p.exists():
            migration_file = p
            break

    if not migration_file:
        print(f"Migration file not found in any of: {[str(p) for p in possible_paths]}")
        return False

    with open(migration_file, 'r') as f:
        sql = f.read()

    # Connect to database and execute migration
    conn_string = settings.get_connection_string()

    try:
        with psycopg.connect(conn_string) as conn:
            with conn.cursor() as cur:
                cur.execute(sql)
                conn.commit()
                print("✓ RAG database schema initialized successfully")
                return True
    except Exception as e:
        print(f"✗ Failed to initialize database: {e}")
        return False


if __name__ == "__main__":
    success = setup_database()
    sys.exit(0 if success else 1)
