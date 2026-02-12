"""Pytest configuration and fixtures for gll tests."""

from pathlib import Path

import pytest


@pytest.fixture
def testdata_dir() -> Path:
    """Return the path to the testdata directory."""
    # Go up from python/tests to root, then into testdata/gll
    return Path(__file__).parent.parent.parent / "testdata" / "gll"


@pytest.fixture
def sample_gll(testdata_dir: Path) -> Path:
    """Return path to a sample GLL file for testing."""
    # Use a smaller file for faster tests
    path = testdata_dir / "D12-v10.gll"
    if path.exists():
        return path
    # Fallback to any available file
    for f in testdata_dir.glob("*.gll"):
        if f.stat().st_size < 5_000_000:  # Prefer files under 5MB
            return f
    pytest.skip("No suitable GLL test file found")


@pytest.fixture
def line_array_gll(testdata_dir: Path) -> Path:
    """Return path to a line array GLL file for testing."""
    path = testdata_dir / "CoRay4-V1_5.gll"
    if path.exists():
        return path
    pytest.skip("Line array GLL test file not found")


@pytest.fixture
def large_gll(testdata_dir: Path) -> Path:
    """Return path to a larger GLL file for comprehensive testing."""
    path = testdata_dir / "APS-V1_1.gll"
    if path.exists():
        return path
    pytest.skip("Large GLL test file not found")
