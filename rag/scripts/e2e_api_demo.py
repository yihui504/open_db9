from __future__ import annotations

"""Run an end-to-end RAG demo against the local API."""

import argparse
import json
import os
import sys
from pathlib import Path

import httpx
from dotenv import load_dotenv

sys.path.insert(0, str(Path(__file__).parent.parent))

from src.config.settings import RAGSettings
from src.dev.bootstrap import ensure_local_context


def load_local_environment() -> None:
    root = Path(__file__).resolve().parents[2]
    env_candidates = [
        root / "deployments" / "docker" / ".env",
        root / "deployments" / "docker" / ".env.example",
    ]

    for env_file in env_candidates:
        if env_file.exists():
            load_dotenv(env_file, override=False)

    mappings = {
        "DB_USER": "RAG_DB_USER",
        "DB_PASSWORD": "RAG_DB_PASSWORD",
        "DB_NAME": "RAG_DB_NAME",
        "JWT_SECRET": "RAG_JWT_SECRET",
    }
    for source, target in mappings.items():
        if os.getenv(source) and not os.getenv(target):
            os.environ[target] = os.environ[source]

    os.environ.setdefault("RAG_DB_HOST", "localhost")
    os.environ.setdefault("RAG_DB_PORT", "5432")


def build_parser() -> argparse.ArgumentParser:
    root = Path(__file__).resolve().parents[2]
    parser = argparse.ArgumentParser()
    parser.add_argument("--api-url", default="http://localhost:8001")
    parser.add_argument("--file", default=str(root / "README.md"))
    parser.add_argument("--question", default="Open-DB9 的核心能力是什么？")
    parser.add_argument("--tenant-name", default="Local Tenant")
    parser.add_argument("--tenant-slug", default="local-tenant")
    parser.add_argument("--database-name", default="Local DB")
    parser.add_argument("--database-slug", default="local-db")
    parser.add_argument("--use-tenant-header", action="store_true")
    return parser


def build_headers(token: str, tenant_id: str, use_tenant_header: bool) -> dict[str, str]:
    headers: dict[str, str] = {}
    if use_tenant_header:
        headers["X-Tenant-ID"] = tenant_id
        return headers
    headers["Authorization"] = f"Bearer {token}"
    return headers


def main() -> int:
    args = build_parser().parse_args()
    load_local_environment()
    settings = RAGSettings(_env_file=None)
    context = ensure_local_context(
        settings=settings,
        tenant_name=args.tenant_name,
        tenant_slug=args.tenant_slug,
        database_name=args.database_name,
        database_slug=args.database_slug,
    )
    headers = build_headers(context.token, context.tenant_id, args.use_tenant_header)
    document_path = Path(args.file).resolve()
    metadata = {"source": document_path.name, "demo": "e2e"}

    print("=== Open-DB9 RAG E2E Demo ===")
    print(f"tenant_id: {context.tenant_id}")
    print(f"database_id: {context.database_id}")
    print(f"file: {document_path}")
    print(f"question: {args.question}")

    with httpx.Client(timeout=120.0) as client:
        with document_path.open("rb") as handle:
            ingest_response = client.post(
                f"{args.api_url}/api/v1/databases/{context.database_id}/rag/documents",
                headers=headers,
                files={"file": (document_path.name, handle, "application/octet-stream")},
                data={
                    "title": document_path.stem,
                    "metadata": json.dumps(metadata, ensure_ascii=False),
                },
            )
        ingest_response.raise_for_status()
        ingest_result = ingest_response.json()
        print(f"document_id: {ingest_result['document_id']}")

        query_response = client.post(
            f"{args.api_url}/api/v1/databases/{context.database_id}/rag/query",
            headers=headers,
            json={"question": args.question, "k": 4},
        )
        query_response.raise_for_status()
        query_result = query_response.json()

    print("")
    print("answer:")
    print(query_result["answer"])
    print("")
    print(f"sources: {len(query_result['sources'])}")
    print(f"latency_ms: {query_result['latency_ms']:.2f}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
