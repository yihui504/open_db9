"""Document parsers for PDF, TXT, and MD files."""

from abc import ABC, abstractmethod
from pathlib import Path
from typing import Dict, Any, List
import pypdf


class DocumentParser(ABC):
    """Base class for document parsers."""

    @abstractmethod
    def parse(self, file_path: str) -> Dict[str, Any]:
        """Parse document and return content with metadata."""
        pass

    @abstractmethod
    def supported_extensions(self) -> List[str]:
        """Return list of supported file extensions."""
        pass


class PDFParser(DocumentParser):
    """Parser for PDF documents."""

    def parse(self, file_path: str) -> Dict[str, Any]:
        """Extract text from PDF."""
        text_content = []
        metadata = {
            "source": file_path,
            "pages": 0,
        }

        with open(file_path, "rb") as f:
            reader = pypdf.PdfReader(f)
            metadata["pages"] = len(reader.pages)

            for page in reader.pages:
                text_content.append(page.extract_text())

        return {
            "content": "\n\n".join(text_content),
            "metadata": metadata,
        }

    def supported_extensions(self) -> List[str]:
        return [".pdf"]


class TextParser(DocumentParser):
    """Parser for plain text documents."""

    def parse(self, file_path: str) -> Dict[str, Any]:
        """Read text file."""
        with open(file_path, "r", encoding="utf-8") as f:
            content = f.read()

        return {
            "content": content,
            "metadata": {"source": file_path},
        }

    def supported_extensions(self) -> List[str]:
        return [".txt", ".md"]


def get_parser(file_path: str) -> DocumentParser:
    """Get appropriate parser for file."""
    ext = Path(file_path).suffix.lower()

    parsers = [PDFParser(), TextParser()]
    for parser in parsers:
        if ext in parser.supported_extensions():
            return parser

    raise ValueError(f"No parser found for extension: {ext}")
