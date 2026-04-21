"""Optional NumPy helpers for converting gll data structures."""

from collections.abc import Mapping
from pathlib import Path
from typing import TYPE_CHECKING, Any

from . import _ffi
from .types import BalloonData, TransferFunction

try:
    import numpy as np
except ImportError:
    np = None
    HAS_NUMPY = False
else:
    HAS_NUMPY = True

if TYPE_CHECKING:
    import numpy as np


if HAS_NUMPY:

    class _ZeroCopyFloat64Array(np.ndarray):  # type: ignore[misc,name-defined]
        """NumPy view that keeps a shared-library buffer alive."""

        _gll_owner: _ffi.RawFloat64Array | None

        def __new__(
            cls, raw: _ffi.RawFloat64Array, shape: tuple[int, int]
        ) -> "_ZeroCopyFloat64Array":
            arr = np.ctypeslib.as_array(raw.pointer, shape=shape).view(cls)
            arr._gll_owner = raw
            return arr

        def __array_finalize__(self, obj: Any) -> None:
            self._gll_owner = getattr(obj, "_gll_owner", None)


def _require_numpy() -> Any:
    """Return the NumPy module or raise a helpful error."""
    if np is None:
        raise ImportError(
            "NumPy support is optional. Install it with: pip install gll[numpy]"
        )
    return np


def transfer_function_to_numpy(tf: TransferFunction) -> tuple[Any, Any, Any]:
    """Convert a transfer function to NumPy arrays.

    Returns:
        Tuple of ``(frequencies_hz, level_db, phase_rad)`` arrays.
    """
    np_mod = _require_numpy()
    return (
        np_mod.asarray(tf.frequencies, dtype=np_mod.float64),
        np_mod.asarray(tf.level, dtype=np_mod.float64),
        np_mod.asarray(tf.phase, dtype=np_mod.float64),
    )


def balloon_grid_to_numpy(balloon: Mapping[str, Any]) -> Any:
    """Convert ``GllFile.get_balloon_at_frequency()`` output to a 2D array."""
    np_mod = _require_numpy()
    return np_mod.asarray(balloon.get("data", []), dtype=np_mod.float64)


def balloon_responses_to_numpy(balloon: BalloonData) -> tuple[Any, Any, Any]:
    """Convert balloon transfer-function responses to NumPy arrays.

    Returns:
        Tuple of ``(frequencies_hz, level_db, phase_rad)`` where level and phase
        are 2D arrays with shape ``(response_count, band_count)``.
    """
    np_mod = _require_numpy()

    if not balloon.responses:
        empty_1d = np_mod.empty((0,), dtype=np_mod.float64)
        empty_2d = np_mod.empty((0, 0), dtype=np_mod.float64)
        return empty_1d, empty_2d, empty_2d

    band_count = len(balloon.responses[0].level)
    phase_count = len(balloon.responses[0].phase)
    for idx, response in enumerate(balloon.responses):
        if len(response.level) != band_count or len(response.phase) != phase_count:
            raise ValueError(f"balloon response {idx} has inconsistent band lengths")

    frequencies = np_mod.asarray(balloon.responses[0].frequencies, dtype=np_mod.float64)
    levels = np_mod.asarray(
        [response.level for response in balloon.responses],
        dtype=np_mod.float64,
    )
    phases = np_mod.asarray(
        [response.phase for response in balloon.responses],
        dtype=np_mod.float64,
    )
    return frequencies, levels, phases


def balloon_to_numpy(balloon: BalloonData | Mapping[str, Any]) -> Any:
    """Convert balloon data to a NumPy array.

    ``BalloonData`` returns a stacked level matrix with shape
    ``(response_count, band_count)``. The dictionary returned by
    ``GllFile.get_balloon_at_frequency()`` returns the angular SPL grid with
    shape ``(meridians, parallels)``.
    """
    if isinstance(balloon, BalloonData):
        _, levels, _ = balloon_responses_to_numpy(balloon)
        return levels
    return balloon_grid_to_numpy(balloon)


def balloon_grid_view(
    source: Any, source_index: int, frequency_hz: float
) -> Any:
    """Create a zero-copy NumPy view over a balloon SPL grid.

    ``source`` may be a ``GllFile`` instance or a filesystem path to a GLL file.
    The returned array has shape ``(meridians, parallels)``.
    """
    _require_numpy()

    if isinstance(source, Path):
        path = source
    elif isinstance(source, str):
        path = Path(source)
    else:
        path = getattr(source, "path", None)
        if path is None:
            raise ValueError("balloon_grid_view requires a GllFile parsed from a path")

    raw = _ffi.get_balloon_grid_raw(path, source_index, frequency_hz)
    return _ZeroCopyFloat64Array(raw, raw.shape)


__all__ = [
    "HAS_NUMPY",
    "balloon_grid_to_numpy",
    "balloon_grid_view",
    "balloon_responses_to_numpy",
    "balloon_to_numpy",
    "transfer_function_to_numpy",
]
