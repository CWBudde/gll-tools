"""CFFI interface to the gll shared library."""

import json
import os
from ctypes import (
    CDLL,
    POINTER,
    Structure,
    c_char_p,
    c_int32,
    c_int64,
    c_double,
    c_void_p,
    string_at,
)
from pathlib import Path
from typing import Any

from .exceptions import GllError, ParseError, ResourceError


class GLL_Result(Structure):
    """Result structure from GLL library calls."""

    _fields_ = [
        ("data", c_char_p),
        ("error", c_char_p),
        ("length", c_int64),
    ]


class GLL_ByteResult(Structure):
    """Binary result structure from GLL library calls."""

    _fields_ = [
        ("data", c_void_p),
        ("length", c_int64),
        ("error", c_char_p),
    ]


def _find_library() -> Path:
    """Find the gll shared library."""
    # Look in the package directory first
    pkg_dir = Path(__file__).parent
    candidates = [
        pkg_dir / "_libgll.so",
        pkg_dir / "_libgll.dylib",
        pkg_dir / "_libgll.dll",
        pkg_dir / "libgll.so",
        pkg_dir / "libgll.dylib",
        pkg_dir / "libgll.dll",
    ]

    for candidate in candidates:
        if candidate.exists():
            return candidate

    # Check environment variable
    env_path = os.environ.get("GLL_LIBRARY_PATH")
    if env_path:
        env_lib = Path(env_path)
        if env_lib.exists():
            return env_lib

    raise RuntimeError(
        f"Could not find gll shared library. Checked: {[str(c) for c in candidates]}"
    )


def _load_library() -> CDLL:
    """Load the gll shared library."""
    lib_path = _find_library()
    lib = CDLL(str(lib_path))

    # Define function signatures
    lib.GLL_Version.argtypes = []
    lib.GLL_Version.restype = c_char_p

    lib.GLL_ParseFile.argtypes = [c_char_p]
    lib.GLL_ParseFile.restype = GLL_Result

    lib.GLL_ParseBytes.argtypes = [c_char_p, c_int64]
    lib.GLL_ParseBytes.restype = GLL_Result

    lib.GLL_ExtractResource.argtypes = [c_char_p, c_int32]
    lib.GLL_ExtractResource.restype = GLL_ByteResult

    lib.GLL_ExtractDataFile.argtypes = [c_char_p, c_int32]
    lib.GLL_ExtractDataFile.restype = GLL_ByteResult

    lib.GLL_ExtractIncludeFile.argtypes = [c_char_p, c_int32]
    lib.GLL_ExtractIncludeFile.restype = GLL_ByteResult

    lib.GLL_ComputeArrayResponse.argtypes = [c_char_p]
    lib.GLL_ComputeArrayResponse.restype = GLL_Result

    lib.GLL_GetBalloonAtFrequency.argtypes = [c_char_p, c_int32, c_double]
    lib.GLL_GetBalloonAtFrequency.restype = GLL_Result

    lib.GLL_FreeResult.argtypes = [GLL_Result]
    lib.GLL_FreeResult.restype = None

    lib.GLL_FreeByteResult.argtypes = [GLL_ByteResult]
    lib.GLL_FreeByteResult.restype = None

    lib.GLL_FreeString.argtypes = [c_char_p]
    lib.GLL_FreeString.restype = None

    return lib


# Module-level library instance (lazy loaded)
_lib: CDLL | None = None


def _get_lib() -> CDLL:
    """Get the library instance, loading it if necessary."""
    global _lib
    if _lib is None:
        _lib = _load_library()
    return _lib


def get_version() -> str:
    """Get the library version string."""
    lib = _get_lib()
    result = lib.GLL_Version()
    if result:
        return result.decode("utf-8")
    return "unknown"


def _check_result(result: GLL_Result) -> dict[str, Any]:
    """Check a GLL_Result and return parsed JSON or raise exception."""
    lib = _get_lib()
    try:
        if result.error:
            error_msg = result.error.decode("utf-8")
            raise ParseError(error_msg)

        if not result.data:
            raise ParseError("No data returned")

        json_str = result.data.decode("utf-8")
        return json.loads(json_str)
    finally:
        lib.GLL_FreeResult(result)


def _check_bytes_result(result: GLL_ByteResult) -> bytes:
    """Check a GLL_ByteResult and return bytes or raise exception."""
    lib = _get_lib()
    try:
        if result.error:
            error_msg = result.error.decode("utf-8")
            raise ResourceError(error_msg)

        if result.data is None or result.length <= 0:
            return b""

        return string_at(result.data, result.length)
    finally:
        lib.GLL_FreeByteResult(result)


def parse_file(path: str | Path) -> dict[str, Any]:
    """Parse a GLL file and return the parsed data as a dictionary."""
    lib = _get_lib()
    path_bytes = str(path).encode("utf-8")
    result = lib.GLL_ParseFile(path_bytes)
    return _check_result(result)


def parse_bytes(data: bytes) -> dict[str, Any]:
    """Parse GLL data from bytes and return the parsed data as a dictionary."""
    lib = _get_lib()
    result = lib.GLL_ParseBytes(data, len(data))
    return _check_result(result)


def extract_resource(path: str | Path, resource_index: int) -> bytes:
    """Extract a resource from a GLL file."""
    lib = _get_lib()
    path_bytes = str(path).encode("utf-8")
    result = lib.GLL_ExtractResource(path_bytes, resource_index)
    return _check_bytes_result(result)


def extract_data_file(path: str | Path, data_file_index: int) -> bytes:
    """Extract a data file from a GLL file."""
    lib = _get_lib()
    path_bytes = str(path).encode("utf-8")
    result = lib.GLL_ExtractDataFile(path_bytes, data_file_index)
    return _check_bytes_result(result)


def extract_include_file(path: str | Path, include_file_index: int) -> bytes:
    """Extract an include file from a GLL file."""
    lib = _get_lib()
    path_bytes = str(path).encode("utf-8")
    result = lib.GLL_ExtractIncludeFile(path_bytes, include_file_index)
    return _check_bytes_result(result)


def compute_array_response(config_json: str) -> dict[str, Any]:
    """Compute array response from JSON configuration."""
    lib = _get_lib()
    config_bytes = config_json.encode("utf-8")
    result = lib.GLL_ComputeArrayResponse(config_bytes)
    return _check_result(result)


def get_balloon_at_frequency(
    path: str | Path, source_index: int, frequency_hz: float
) -> dict[str, Any]:
    """Get balloon data at a specific frequency."""
    lib = _get_lib()
    path_bytes = str(path).encode("utf-8")
    result = lib.GLL_GetBalloonAtFrequency(path_bytes, source_index, frequency_hz)
    return _check_result(result)
