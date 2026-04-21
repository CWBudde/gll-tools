"""Tests for zero-copy NumPy balloon views."""

from pathlib import Path

import pytest

from gll import GllFile, HAS_NUMPY, balloon_grid_view


def test_balloon_grid_view_requires_numpy(sample_gll: Path) -> None:
    """Zero-copy view should fail clearly without NumPy."""
    if HAS_NUMPY:
        pytest.skip("NumPy is available")

    with pytest.raises(ImportError):
        balloon_grid_view(sample_gll, 0, 1000.0)


def test_balloon_grid_view(sample_gll: Path) -> None:
    """Balloon grid view returns a 2D NumPy array when available."""
    if not HAS_NUMPY:
        pytest.skip("NumPy is not available")

    gll = GllFile.parse(sample_gll)
    sources = gll.database.source_definitions
    if not sources:
        pytest.skip("No source definitions")

    has_balloon = any(
        source.balloon_data is not None and source.balloon_data.responses
        for source in sources
    )
    if not has_balloon:
        pytest.skip("No balloon data in source definitions")

    view = balloon_grid_view(gll, 0, 1000.0)
    assert view.ndim == 2
    assert view.shape[0] > 0
    assert view.shape[1] > 0
    assert view.dtype.name == "float64"
