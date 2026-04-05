"""Embeddings wrapper for cloud and local providers."""

from langchain_core.embeddings import Embeddings
from typing import List, Optional
import hashlib
import math
import re


class LocalHashEmbeddings(Embeddings):
    def __init__(self, dimension: int = 1024):
        self._dimension = dimension

    def _embed(self, text: str) -> List[float]:
        vector = [0.0] * self._dimension
        tokens = re.findall(r"\w+", text.lower())

        if not tokens:
            return vector

        for token in tokens:
            digest = hashlib.blake2b(token.encode("utf-8"), digest_size=16).digest()
            index = int.from_bytes(digest[:8], "big") % self._dimension
            weight = 1.0 + (digest[8] / 255.0)
            vector[index] += weight

        norm = math.sqrt(sum(value * value for value in vector))
        if norm == 0:
            return vector

        return [value / norm for value in vector]

    def embed_documents(self, texts: List[str]) -> List[List[float]]:
        return [self._embed(text) for text in texts]

    def embed_query(self, text: str) -> List[float]:
        return self._embed(text)


class DB9Embeddings(Embeddings):
    """Wrapper for embeddings with caching and batch optimization."""

    def __init__(
        self,
        provider: str = "local",
        model: Optional[str] = None,
        api_key: str = "",
        batch_size: int = 100,
        dimension: int = 1024,
    ):
        if provider == "zhipuai":
            from langchain_community.embeddings import ZhipuAIEmbeddings
            self._embeddings = ZhipuAIEmbeddings(
                model=model or "embedding-2",
                api_key=api_key,
            )
        elif provider == "openai":
            from langchain_openai import OpenAIEmbeddings
            self._embeddings = OpenAIEmbeddings(
                model=model or "text-embedding-3-small",
                openai_api_key=api_key,
            )
        else:
            self._embeddings = LocalHashEmbeddings(dimension=dimension)
        self._batch_size = batch_size
        self._cache = {}

    def embed_documents(self, texts: List[str]) -> List[List[float]]:
        """Embed search docs."""
        # Process in batches
        results = []
        for i in range(0, len(texts), self._batch_size):
            batch = texts[i:i + self._batch_size]
            batch_results = self._embeddings.embed_documents(batch)
            results.extend(batch_results)
        return results

    def embed_query(self, text: str) -> List[float]:
        """Embed query text."""
        return self._embeddings.embed_query(text)
