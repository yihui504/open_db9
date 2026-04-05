from __future__ import annotations

"""Bootstrap local tenant, database, and JWT for RAG development."""

import argparse
import os
import sys
from pathlib import Path

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
    parser = argparse.ArgumentParser()
    parser.add_argument("--tenant-name", default="Local Tenant")
    parser.add_argument("--tenant-slug", default="local-tenant")
    parser.add_argument("--database-name", default="Local DB")
    parser.add_argument("--database-slug", default="local-db")
    parser.add_argument("--username", default="local-dev")
    parser.add_argument("--user-id", type=int, default=1)
    parser.add_argument("--json", action="store_true")
    return parser


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
        username=args.username,
        user_id=args.user_id,
    )

    if args.json:
        import json

        print(
            json.dumps(
                {
                    "tenant_id": context.tenant_id,
                    "database_id": context.database_id,
                    "token": context.token,
                },
                ensure_ascii=False,
                indent=2,
            )
        )
        return 0

    print(f"tenant_id={context.tenant_id}")
    print(f"database_id={context.database_id}")
    print(f"token={context.token}")
    print(
        "query_example="
        f"python -m rag.src.cli.commands --database-id {context.database_id} "
        f"--api-token {context.token} query \"Open-DB9 是什么？\""
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
