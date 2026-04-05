"""FastAPI server for RAG system."""

from fastapi import FastAPI
from .routes import router
from ..config.settings import get_settings

settings = get_settings()

app = FastAPI(
    title="Open-DB9 RAG API",
    description="Retrieval-Augmented Generation API for Open-DB9",
    version="0.1.0",
)

# Include RAG routes
app.include_router(router, prefix="/api/v1")


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy", "service": "rag-api"}


@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "message": "Open-DB9 RAG API",
        "version": "0.1.0",
        "docs": "/docs",
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "src.api.server:app",
        host=settings.api_host,
        port=settings.api_port,
        reload=True,
    )
