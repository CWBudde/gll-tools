"""Tests for resource extraction."""

from pathlib import Path

import pytest

from gll import GllFile, GllError


def test_list_resources(sample_gll: Path) -> None:
    """Test listing resources."""
    gll = GllFile.parse(sample_gll)
    resources = gll.resources
    assert isinstance(resources, list)
    # Resources may be empty for some files


def test_extract_resource(sample_gll: Path) -> None:
    """Test extracting a resource."""
    gll = GllFile.parse(sample_gll)
    resources = gll.resources
    if not resources:
        pytest.skip("No resources in test file")

    data = gll.extract_resource(resources[0])
    assert isinstance(data, bytes)
    assert len(data) > 0


def test_extract_resource_by_index(sample_gll: Path) -> None:
    """Test extracting a resource by index."""
    gll = GllFile.parse(sample_gll)
    resources = gll.resources
    if not resources:
        pytest.skip("No resources in test file")

    data = gll.extract_resource(0)
    assert isinstance(data, bytes)
    assert len(data) > 0


def test_extract_resource_from_bytes_fails(sample_gll: Path) -> None:
    """Test that extracting from bytes-parsed file fails."""
    with open(sample_gll, "rb") as f:
        gll = GllFile.parse(f)

    with pytest.raises(GllError):
        gll.extract_resource(0)


def test_list_data_files(sample_gll: Path) -> None:
    """Test listing data files."""
    gll = GllFile.parse(sample_gll)
    data_files = gll.database.data_files
    assert isinstance(data_files, list)


def test_extract_data_file(sample_gll: Path) -> None:
    """Test extracting a data file."""
    gll = GllFile.parse(sample_gll)
    data_files = gll.database.data_files
    if not data_files:
        pytest.skip("No data files in test file")

    data = gll.extract_data_file(data_files[0])
    assert isinstance(data, bytes)
    assert len(data) > 0


def test_list_include_files(sample_gll: Path) -> None:
    """Test listing include files."""
    gll = GllFile.parse(sample_gll)
    include_files = gll.database.include_files
    assert isinstance(include_files, list)


def test_extract_include_file(large_gll: Path) -> None:
    """Test extracting an include file (large files often have these)."""
    gll = GllFile.parse(large_gll)
    include_files = gll.database.include_files
    if not include_files:
        pytest.skip("No include files in test file")

    data = gll.extract_include_file(include_files[0])
    assert isinstance(data, bytes)
    assert len(data) > 0


def test_resource_properties(sample_gll: Path) -> None:
    """Test resource property access."""
    gll = GllFile.parse(sample_gll)
    resources = gll.resources
    if not resources:
        pytest.skip("No resources in test file")

    res = resources[0]
    assert isinstance(res.index, int)
    assert isinstance(res.type, str)
    assert isinstance(res.offset, int)
    assert isinstance(res.size, int)


def test_data_file_properties(sample_gll: Path) -> None:
    """Test data file property access."""
    gll = GllFile.parse(sample_gll)
    data_files = gll.database.data_files
    if not data_files:
        pytest.skip("No data files in test file")

    df = data_files[0]
    assert isinstance(df.index, int)
    assert isinstance(df.key, str)
    assert isinstance(df.filename, str)
    assert isinstance(df.size, int)
