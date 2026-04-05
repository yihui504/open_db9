"""Configuration management for RAG system."""

from pydantic_settings import BaseSettings
from functools import lru_cache
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from langchain_core.embeddings import Embeddings


class RAGSettings(BaseSettings):
    """RAG system configuration."""

    # Database
    db_host: str = "localhost"
    db_port: int = 5432
    db_user: str = "postgres"
    db_password: str = ""
    db_name: str = "db9"

    # Open-DB9 API
    db9_api_url: str = "http://localhost:8080"
    db9_api_token: str = ""

    # FS9 Service
    fs9_url: str = "http://localhost:9090"
    fs9_upload_timeout: int = 300

    provider: str = "zhipuai"

    embedding_model: str = "text-embedding-3-small"
    embedding_dimension: int = 1024
    openai_api_key: str = ""

    zhipuai_api_key: str = ""
    zhipuai_embedding_model: str = "embedding-2"
    zhipuai_llm_model: str = "glm-4"
    local_embedding_model: str = "local-hash"
    local_embedding_dimension: int = 1024

    chunk_size: int = 1000
    chunk_overlap: int = 200

    default_k: int = 4
    similarity_threshold: float = 0.7

    llm_model: str = "gpt-3.5-turbo"
    llm_temperature: float = 0.7

    api_host: str = "0.0.0.0"
    api_port: int = 8001

    jwt_secret: str = ""
    jwt_algorithm: str = "HS256"

    class Config:
        env_prefix = "RAG_"
        env_file = ".env"
        extra = "ignore"

    def get_connection_string(self) -> str:
        """Build and return PostgreSQL connection string."""
        return f"postgresql://{self.db_user}:{self.db_password}@{self.db_host}:{self.db_port}/{self.db_name}"

    def get_embedding_dimension(self) -> int:
        if self.provider == "openai":
            return 1536
        if self.provider == "zhipuai":
            return 1024
        return self.local_embedding_dimension

    def _require_api_key(self, provider: str, api_key: str, env_name: str) -> None:
        if self.provider == provider and not api_key:
            raise ValueError(f"{env_name} must be set when RAG_PROVIDER={provider}")

    def get_embeddings(self) -> "Embeddings":
        """Get configured Embeddings instance."""
        if self.provider == "zhipuai":
            self._require_api_key("zhipuai", self.zhipuai_api_key, "RAG_ZHIPUAI_API_KEY")
            from ..vectorstore.embeddings import DB9Embeddings
            return DB9Embeddings(
                provider="zhipuai",
                model=self.zhipuai_embedding_model,
                api_key=self.zhipuai_api_key
            )

        if self.provider == "openai":
            self._require_api_key("openai", self.openai_api_key, "RAG_OPENAI_API_KEY")
            from ..vectorstore.embeddings import DB9Embeddings
            return DB9Embeddings(
                provider="openai",
                model=self.embedding_model,
                api_key=self.openai_api_key
            )

        from ..vectorstore.embeddings import DB9Embeddings
        return DB9Embeddings(
            provider="local",
            model=self.local_embedding_model,
            dimension=self.local_embedding_dimension,
        )


@lru_cache()
def get_settings() -> RAGSettings:
    return RAGSettings()
