"""Tests for optional NumPy compatibility helpers."""

import pytest

from gll import (
    AngularResolution,
    BalloonData,
    HAS_NUMPY,
    TransferFunction,
    TransferFunctionDef,
    balloon_grid_to_numpy,
    balloon_responses_to_numpy,
    balloon_to_numpy,
    transfer_function_to_numpy,
)


def test_transfer_function_to_numpy() -> None:
    """Transfer functions convert to arrays when NumPy is available."""
    tf = TransferFunction(
        definition=TransferFunctionDef(
            start_frequency=100.0,
            end_frequency=400.0,
            bands_per_octave=1,
        ),
        level=[1.0, 2.0],
        phase=[0.1, 0.2],
    )

    if not HAS_NUMPY:
        with pytest.raises(ImportError):
            transfer_function_to_numpy(tf)
        return

    freqs, level, phase = transfer_function_to_numpy(tf)
    assert freqs.tolist() == [100.0, 200.0]
    assert level.tolist() == [1.0, 2.0]
    assert phase.tolist() == [0.1, 0.2]


def test_balloon_grid_to_numpy() -> None:
    """Balloon grid dictionaries convert to 2D arrays."""
    balloon = {
        "frequency": 1000.0,
        "data": [[1.0, 2.0], [3.0, 4.0]],
    }

    if not HAS_NUMPY:
        with pytest.raises(ImportError):
            balloon_grid_to_numpy(balloon)
        return

    grid = balloon_grid_to_numpy(balloon)
    assert grid.shape == (2, 2)
    assert grid.tolist() == [[1.0, 2.0], [3.0, 4.0]]


def test_balloon_responses_to_numpy() -> None:
    """Balloon response sets convert to stacked NumPy matrices."""
    balloon = BalloonData(
        angular_resolution=AngularResolution(),
        responses=[
            TransferFunction(
                definition=TransferFunctionDef(
                    start_frequency=100.0,
                    end_frequency=400.0,
                    bands_per_octave=1,
                ),
                level=[1.0, 2.0],
                phase=[0.1, 0.2],
            ),
            TransferFunction(
                definition=TransferFunctionDef(
                    start_frequency=100.0,
                    end_frequency=400.0,
                    bands_per_octave=1,
                ),
                level=[3.0, 4.0],
                phase=[0.3, 0.4],
            ),
        ],
    )

    if not HAS_NUMPY:
        with pytest.raises(ImportError):
            balloon_responses_to_numpy(balloon)
        with pytest.raises(ImportError):
            balloon_to_numpy(balloon)
        return

    freqs, levels, phases = balloon_responses_to_numpy(balloon)
    assert freqs.tolist() == [100.0, 200.0]
    assert levels.shape == (2, 2)
    assert phases.shape == (2, 2)
    assert balloon_to_numpy(balloon).tolist() == [[1.0, 2.0], [3.0, 4.0]]
