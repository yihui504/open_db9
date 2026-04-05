from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from typing import TYPE_CHECKING
from uuid import NAMESPACE_URL, uuid5

from jose import jwt

if TYPE_CHECKING:
    from ..config.settings import RAGSettings


@dataclass
class LocalContext:
    tenant_id: str
    database_id: str
    token: str


def derive_context_ids(tenant_slug: str, database_slug: str) -> tuple[str, str]:
    tenant_id = str(uuid5(NAMESPACE_URL, f"tenant:{tenant_slug}"))
    database_id = str(uuid5(NAMESPACE_URL, f"database:{tenant_slug}:{database_slug}"))
    return tenant_id, database_id


def issue_local_token(
    secret: str,
    tenant_id: str,
    username: str = "local-dev",
    user_id: int = 1,
    expires_in_hours: int = 24,
) -> str:
    if not secret:
        raise ValueError("JWT secret is required to issue local token")

    now = datetime.now(timezone.utc)
    payload = {
        "tenant_id": tenant_id,
        "user_id": user_id,
        "username": username,
        "iss": "open-db9-local",
        "sub": f"user:{user_id}",
        "iat": int(now.timestamp()),
        "nbf": int(now.timestamp()),
        "exp": int((now + timedelta(hours=expires_in_hours)).timestamp()),
    }
    return jwt.encode(payload, secret, algorithm="HS256")


def ensure_local_context(
    settings: "RAGSettings",
    tenant_name: str = "Local Tenant",
    tenant_slug: str = "local-tenant",
    database_name: str = "Local DB",
    database_slug: str = "local-db",
    username: str = "local-dev",
    user_id: int = 1,
) -> LocalContext:
    import psycopg

    tenant_id, database_id = derive_context_ids(tenant_slug, database_slug)

    with psycopg.connect(settings.get_connection_string(), autocommit=True) as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO tenants (id, name, slug)
                VALUES (%s, %s, %s)
                ON CONFLICT (id) DO UPDATE
                SET name = EXCLUDED.name, slug = EXCLUDED.slug
                """,
                [tenant_id, tenant_name, tenant_slug],
            )
            cur.execute(
                """
                INSERT INTO databases (
                    id, tenant_id, name, slug, engine, version, region, size
                )
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (id) DO UPDATE
                SET name = EXCLUDED.name, slug = EXCLUDED.slug
                """,
                [database_id, tenant_id, database_name, database_slug, "postgresql", "16", "local", "small"],
            )

    token = issue_local_token(settings.jwt_secret, tenant_id, username=username, user_id=user_id)
    return LocalContext(tenant_id=tenant_id, database_id=database_id, token=token)
