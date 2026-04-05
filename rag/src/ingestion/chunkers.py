"""Text chunking strategies for document processing."""

from typing import List, Dict, Any
from langchain_text_splitters import (
    RecursiveCharacterTextSplitter,
    MarkdownTextSplitter,
)


class DocumentChunker:
    """Chunk documents for embedding."""

    def __init__(
        self,
        chunk_size: int = 1000,
        chunk_overlap: int = 200,
        is_markdown: bool = False,
    ):
        self.chunk_size = chunk_size
        self.chunk_overlap = chunk_overlap
        self.is_markdown = is_markdown

        if is_markdown:
            self.splitter = MarkdownTextSplitter(
                chunk_size=chunk_size,
                chunk_overlap=chunk_overlap,
            )
        else:
            self.splitter = RecursiveCharacterTextSplitter(
                chunk_size=chunk_size,
                chunk_overlap=chunk_overlap,
                separators=["\n\n", "\n", ". ", " ", ""],
            )

    def chunk(
        self,
        content: str,
        metadata: Dict[str, Any],
    ) -> List[Dict[str, Any]]:
        """Split content into chunks."""
        chunks = self.splitter.split_text(content)

        result = []
        for i, chunk in enumerate(chunks):
            chunk_metadata = {
                **metadata,
                "chunk_index": i,
                "chunk_size": len(chunk),
            }
            result.append({
                "content": chunk,
                "metadata": chunk_metadata,
            })

        return result
