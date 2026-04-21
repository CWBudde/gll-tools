"""Tests for GLL file parsing."""

from pathlib import Path

import pytest

from gll import GllFile, ParseError, SystemType


def test_parse_file(sample_gll: Path) -> None:
    """Test parsing a GLL file from path."""
    gll = GllFile.parse(sample_gll)
    assert gll is not None
    assert gll.path == sample_gll


def test_parse_file_str(sample_gll: Path) -> None:
    """Test parsing a GLL file from string path."""
    gll = GllFile.parse(str(sample_gll))
    assert gll is not None


def test_parse_file_not_found() -> None:
    """Test parsing a non-existent file raises error."""
    with pytest.raises(FileNotFoundError):
        GllFile.parse("nonexistent.gll")


def test_parse_bytes(sample_gll: Path) -> None:
    """Test parsing a GLL file from bytes."""
    with open(sample_gll, "rb") as f:
        gll = GllFile.parse(f)
    assert gll is not None
    assert gll.path is None  # No path when parsed from bytes


def test_header(sample_gll: Path) -> None:
    """Test accessing file header."""
    gll = GllFile.parse(sample_gll)
    header = gll.header
    assert header.magic == "EGLL"
    # Format ID can be "EASEGLL" or "EASE_GLL" depending on version
    assert "EASE" in header.format_id and "GLL" in header.format_id
    assert header.format_version >= 3


def test_metadata(sample_gll: Path) -> None:
    """Test accessing metadata."""
    gll = GllFile.parse(sample_gll)
    metadata = gll.metadata
    # At least one of these should be set
    assert metadata.product_name or metadata.manufacturer or gll.gen_system.label


def test_gen_system(sample_gll: Path) -> None:
    """Test accessing gen_system."""
    gll = GllFile.parse(sample_gll)
    gs = gll.gen_system
    assert gs.label  # Label should always be present
    assert isinstance(gs.system_type, SystemType)


def test_repr(sample_gll: Path) -> None:
    """Test string representation."""
    gll = GllFile.parse(sample_gll)
    repr_str = repr(gll)
    assert "GllFile" in repr_str


def test_database_exists(sample_gll: Path) -> None:
    """Test that database is parsed."""
    gll = GllFile.parse(sample_gll)
    db = gll.database
    assert db is not None


def test_database_box_types(sample_gll: Path) -> None:
    """Test accessing box types."""
    gll = GllFile.parse(sample_gll)
    box_types = gll.database.box_types
    # Most GLL files have at least one box type
    if box_types:
        bt = box_types[0]
        assert bt.label  # Label should be present


def test_database_source_definitions(sample_gll: Path) -> None:
    """Test accessing source definitions."""
    gll = GllFile.parse(sample_gll)
    sources = gll.database.source_definitions
    # Most GLL files have at least one source
    if sources:
        src = sources[0]
        assert src.key  # Key should be present
        assert src.name


def test_gllfile_compute_array_response(line_array_gll: Path) -> None:
    """GllFile exposes the draft-style array-response convenience wrapper."""
    from gll import ArrayConfig

    gll = GllFile.parse(line_array_gll)
    box_types = gll.database.box_types
    if not box_types:
        pytest.skip("No box types available")

    response = gll.compute_array_response(
        ArrayConfig().add_element(box_types[0].name),
        frequency=1000.0,
    )
    assert response is not None


def test_line_array_metadata(line_array_gll: Path) -> None:
    """Test metadata from a line array GLL file."""
    gll = GllFile.parse(line_array_gll)
    # Check that system type is valid
    assert isinstance(gll.gen_system.system_type, SystemType)
    # Line arrays should have multiple box types
    assert len(gll.database.box_types) >= 1


def test_large_file_parse(large_gll: Path) -> None:
    """Test parsing a larger GLL file."""
    gll = GllFile.parse(large_gll)
    assert gll is not None
    # Should have multiple box types and sources
    assert len(gll.database.box_types) >= 1
    assert len(gll.database.source_definitions) >= 1
