"""CLI commands for RAG system using Click."""

import click
from pathlib import Path
import httpx


@click.group()
@click.option("--api-url", default="http://localhost:8001", help="RAG API URL")
@click.option("--api-token", help="JWT token for authentication")
@click.option("--database-id", required=True, help="Database ID")
@click.option("--tenant-id", help="Tenant ID for local development")
@click.pass_context
def rag(ctx, api_url, api_token, database_id, tenant_id):
    """RAG system CLI commands."""
    ctx.ensure_object(dict)
    ctx.obj["api_url"] = api_url
    ctx.obj["api_token"] = api_token
    ctx.obj["database_id"] = database_id
    ctx.obj["tenant_id"] = tenant_id


@rag.command()
@click.argument("file_path", type=click.Path(exists=True))
@click.option("--title", help="Document title")
@click.option("--metadata", help="Metadata as JSON string")
@click.pass_context
def ingest(ctx, file_path, title, metadata):
    """Ingest a document into the RAG system."""
    import json

    api_url = ctx.obj["api_url"]
    database_id = ctx.obj["database_id"]
    headers = {}
    if ctx.obj.get("api_token"):
        headers["Authorization"] = f"Bearer {ctx.obj['api_token']}"
    if ctx.obj.get("tenant_id"):
        headers["X-Tenant-ID"] = ctx.obj["tenant_id"]

    with open(file_path, "rb") as f:
        files = {"file": (Path(file_path).name, f)}
        data = {}
        if title:
            data["title"] = title
        if metadata:
            data["metadata"] = metadata

        response = httpx.post(
            f"{api_url}/api/v1/databases/{database_id}/rag/documents",
            files=files,
            data=data,
            headers=headers
        )
        response.raise_for_status()

    result = response.json()
    click.echo(f"Document ingested: {result['document_id']}")


@rag.command()
@click.argument("question")
@click.option("--k", default=4, help="Number of chunks to retrieve")
@click.pass_context
def query(ctx, question, k):
    """Query the RAG system."""
    api_url = ctx.obj["api_url"]
    database_id = ctx.obj["database_id"]
    headers = {}
    if ctx.obj.get("api_token"):
        headers["Authorization"] = f"Bearer {ctx.obj['api_token']}"
    if ctx.obj.get("tenant_id"):
        headers["X-Tenant-ID"] = ctx.obj["tenant_id"]

    payload = {"question": question, "k": k}
    response = httpx.post(
        f"{api_url}/api/v1/databases/{database_id}/rag/query",
        json=payload,
        headers=headers
    )
    response.raise_for_status()

    result = response.json()
    click.echo(f"Question: {result['question']}")
    click.echo(f"Answer: {result['answer']}")
    click.echo(f"\nSources ({len(result['sources'])}):")
    for i, source in enumerate(result["sources"], 1):
        click.echo(f"{i}. {source.get('title', 'Unknown')}")


@rag.command()
@click.pass_context
def list(ctx):
    """List all documents in the RAG system."""
    api_url = ctx.obj["api_url"]
    database_id = ctx.obj["database_id"]
    headers = {}
    if ctx.obj.get("api_token"):
        headers["Authorization"] = f"Bearer {ctx.obj['api_token']}"
    if ctx.obj.get("tenant_id"):
        headers["X-Tenant-ID"] = ctx.obj["tenant_id"]

    response = httpx.get(
        f"{api_url}/api/v1/databases/{database_id}/rag/documents",
        headers=headers
    )
    response.raise_for_status()

    result = response.json()
    click.echo(f"Documents ({result['total']}):")
    for doc in result["documents"]:
        click.echo(f"- {doc['title']} ({doc['total_chunks']} chunks)")


if __name__ == "__main__":
    rag(obj={})
