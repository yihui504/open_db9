"""Unit tests for document parsers."""

import pytest
import sys
import tempfile
from pathlib import Path

# Add src to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "src"))

from ingestion.parsers import PDFParser, TextParser, get_parser


def test_text_parser():
    """Test text parser."""
    parser = TextParser()

    # Create a temporary text file
    with tempfile.NamedTemporaryFile(mode='w', suffix='.txt', delete=False) as f:
        f.write("Test content\nLine 2\nLine 3")
        temp_path = f.name

    try:
        result = parser.parse(temp_path)

        assert result["content"] == "Test content\nLine 2\nLine 3"
        assert result["metadata"]["source"] == temp_path
    finally:
        Path(temp_path).unlink()


def test_pdf_parser_supported_extensions():
    """Test PDF parser supported extensions."""
    parser = PDFParser()
    assert parser.supported_extensions() == [".pdf"]


def test_text_parser_supported_extensions():
    """Test text parser supported extensions."""
    parser = TextParser()
    assert ".txt" in parser.supported_extensions()
    assert ".md" in parser.supported_extensions()


def test_get_parser_txt():
    """Test get_parser for .txt files."""
    with tempfile.NamedTemporaryFile(suffix='.txt', delete=False) as f:
        temp_path = f.name

    try:
        parser = get_parser(temp_path)
        assert isinstance(parser, TextParser)
    finally:
        Path(temp_path).unlink()


def test_get_parser_unsupported():
    """Test get_parser with unsupported extension."""
    with tempfile.NamedTemporaryFile(suffix='.xyz', delete=False) as f:
        temp_path = f.name

    try:
        with pytest.raises(ValueError, match="No parser found"):
            get_parser(temp_path)
    finally:
        Path(temp_path).unlink()
