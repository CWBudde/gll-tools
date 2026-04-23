# Python Bindings for gll-tools - Implementation Plan

This document outlines the long-term plan for creating native Python bindings for the gll-tools library using cgo.

## Overview

**Goal:** Enable Python developers to import and use gll-tools as a native Python library with idiomatic API.

```python
from gll import GllFile

# Parse a GLL file
gll_file = GllFile.parse("speaker.gll")

# Access metadata
print(gll_file.metadata.manufacturer)
print(gll_file.metadata.product_name)

# Iterate over sources
for source in gll_file.database.source_definitions:
    print(f"Source: {source.name}")
    print(f"  Sensitivity: {source.sensitivity_1w_1m} dB")

    # Access balloon data
    balloon = source.get_balloon_at_frequency(1000)
    spl_on_axis = balloon.get_spl(0, 0)  # azimuth=0, elevation=0

# Extract resources
for resource in gll_file.resources:
    data = gll_file.extract_resource(resource)
    with open(resource.name, 'wb') as f:
        f.write(data)

# Array calculations
config = ArrayConfig()
config.add_element(box_type="K2", splay=0.5)
config.add_element(box_type="K2", splay=1.0)

response = gll_file.compute_array_response(config, frequency=1000)
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Python Layer                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   gll.py    │  │  types.py   │  │      exceptions.py      │  │
│  │ (High-level │  │ (Dataclass  │  │   (GllError,            │  │
│  │   API)      │  │  wrappers)  │  │    ParseError, etc.)    │  │
│  └──────┬──────┘  └──────┬──────┘  └────────────┬────────────┘  │
└─────────┼────────────────┼──────────────────────┼───────────────┘
          │                │                      │
          ▼                ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                     CFFI/ctypes Bridge                          │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  _gll_ffi.py - C function declarations & memory management  ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    C Shared Library (libgll.so)                 │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  gll_cgo.go - //export functions with C ABI                │ │
│  │                                                            │ │
│  │  • GLL_Parse(path) -> JSON                                 │ │
│  │  • GLL_ParseBytes(data, len) -> JSON                       │ │
│  │  • GLL_ExtractResource(handle, index) -> bytes             │ │
│  │  • GLL_ComputeArrayResponse(...) -> JSON                   │ │
│  │  • GLL_Free*(ptr) - Memory cleanup functions               │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Go Library (pkg/gll)                        │
│  ┌──────────┐ ┌───────────┐ ┌────────────┐ ┌─────────────────┐  │
│  │ parse.go │ │database.go│ │ extract.go │ │array_calc.go    │  │
│  └──────────┘ └───────────┘ └────────────┘ └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Status Summary

| Phase                       | Status      | Notes                                                                                                                  |
| --------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------- |
| 1. Foundation               | In progress | Core structure and local build flow exist; a few draft convenience files/targets are still missing                     |
| 2. Core Python API          | In progress | Main API is implemented, but not every draft method/module exists exactly as originally sketched                       |
| 3. Acoustic Calculations    | In progress | Array response API works; draft-only extras such as combined balloon output are not implemented                        |
| 4. NumPy Integration        | Done        | Optional NumPy helpers and zero-copy balloon grid view are implemented                                                 |
| 5. Packaging & Distribution | In progress | `pyproject.toml` and local wheel build exist; release automation for Python wheels is still missing                    |
| 6. Testing & Documentation  | In progress | Strong local pytest coverage and README docs exist; dedicated docs tree and memory-safety automation are still missing |

## Phase 1: Foundation (4-6 weeks)

**Checklist**

- [x] `python/` package root exists with `gll/`, `tests/`, `examples/`, `README.md`, and `pyproject.toml`.
- [x] `cmd/gllpy/main.go` exists as the c-shared export entry point.
- [x] Local developer tasks exist for Python build/install/test/lint/typecheck/wheel flows in `justfile`.
- [x] Shared library artifacts are generated into `python/gll/`.
- [x] Dedicated `python/gll/database.py` wrapper module from the original sketch exists as a separate file.
- [x] `python/setup.py` compatibility shim exists.
- [x] A multi-platform `build-python-all` target exists in the repo automation.

**Acceptance Criteria**

- [x] The repository can build `python/gll/_libgll.so` from `./cmd/gllpy`.
- [x] The Python bindings have a coherent package/test/example layout suitable for local development.
- [x] The project has a documented local install path for the Python bindings.
- [ ] All draft structure files and helper targets from the original plan are present verbatim.

### 1.1 Project Structure

Create a new directory structure for Python bindings:

```
gll-tools/
├── python/
│   ├── gll/                    # Python package
│   │   ├── __init__.py
│   │   ├── _ffi.py             # CFFI interface
│   │   ├── types.py            # Python type definitions
│   │   ├── file.py             # GllFile class
│   │   ├── database.py         # Database wrapper classes
│   │   ├── acoustics.py        # Acoustic calculation wrappers
│   │   └── exceptions.py       # Exception hierarchy
│   ├── tests/
│   │   ├── test_parse.py
│   │   ├── test_extract.py
│   │   └── test_acoustics.py
│   ├── examples/
│   │   ├── basic_parse.py
│   │   ├── extract_resources.py
│   │   └── array_response.py
│   ├── pyproject.toml
│   ├── setup.py                # Build script for shared library
│   └── README.md
├── cmd/
│   └── gllpy/                  # cgo export module
│       └── main.go
└── ...
```

### 1.2 cgo Export Layer

Create `cmd/gllpy/main.go` with C-callable exports:

```go
package main

/*
#include <stdlib.h>
#include <stdint.h>

// Result structure for functions that return data + error
typedef struct {
    char* data;      // JSON string or NULL on error
    char* error;     // Error message or NULL on success
    int64_t length;  // Length of binary data (if applicable)
} GLL_Result;
*/
import "C"

import (
    "encoding/json"
    "unsafe"

    "github.com/cwbudde/gll-tools/pkg/gll"
)

// GLL_ParseFile parses a GLL file and returns JSON metadata
//export GLL_ParseFile
func GLL_ParseFile(path *C.char) C.GLL_Result {
    // Implementation
}

// GLL_ParseBytes parses GLL data from memory
//export GLL_ParseBytes
func GLL_ParseBytes(data *C.char, length C.int64_t) C.GLL_Result {
    // Implementation
}

// GLL_ExtractResource extracts a resource by index
//export GLL_ExtractResource
func GLL_ExtractResource(path *C.char, resourceIndex C.int32_t) C.GLL_Result {
    // Implementation
}

// GLL_ComputeArrayResponse calculates combined array response
//export GLL_ComputeArrayResponse
func GLL_ComputeArrayResponse(configJSON *C.char) C.GLL_Result {
    // Implementation
}

// Memory management exports
//export GLL_FreeResult
func GLL_FreeResult(result C.GLL_Result) {
    if result.data != nil {
        C.free(unsafe.Pointer(result.data))
    }
    if result.error != nil {
        C.free(unsafe.Pointer(result.error))
    }
}

//export GLL_Version
func GLL_Version() *C.char {
    return C.CString("0.1.0")
}

func main() {} // Required but unused for c-shared
```

### 1.3 Data Serialization Strategy

**Decision: Use JSON for complex types, raw bytes for binary data**

| Data Type           | Serialization | Rationale                                 |
| ------------------- | ------------- | ----------------------------------------- |
| File metadata       | JSON          | Complex nested structure, rarely hot path |
| Database contents   | JSON          | Same as above                             |
| Balloon data        | JSON arrays   | Numeric arrays serialize well             |
| Transfer functions  | JSON          | Level/Phase arrays                        |
| Extracted resources | Raw bytes     | Binary data (PNG, PDF, etc.)              |
| Error messages      | C string      | Simple strings                            |

### 1.4 Build System

**Makefile additions:**

```makefile
# Build shared library for Python
build-python:
    CGO_ENABLED=1 go build -buildmode=c-shared \
        -o python/gll/_libgll.so \
        ./cmd/gllpy

# Build for multiple platforms
build-python-all:
    # Linux x86_64
    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
        go build -buildmode=c-shared -o dist/libgll-linux-amd64.so ./cmd/gllpy

    # Linux arm64
    GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
        go build -buildmode=c-shared -o dist/libgll-linux-arm64.so ./cmd/gllpy

    # macOS (requires macOS or cross-compiler)
    # Windows (requires mingw)
```

---

## Phase 2: Core Python API (3-4 weeks)

**Checklist**

- [x] `python/gll/_ffi.py` loads the shared library and exposes typed wrappers for parse/extract/acoustic calls.
- [x] `python/gll/types.py` defines typed wrappers for headers, metadata, database content, and acoustic data.
- [x] `python/gll/file.py` implements path-based parsing, bytes parsing, lazy accessors, and extraction helpers.
- [x] `python/gll/exceptions.py` defines Python exception types used by the bindings.
- [x] `python/gll/__init__.py` exports the public package surface.
- [x] The original draft API shape is implemented exactly, including draft-only helper methods such as `BalloonData.get_spl()` and `SourceDefinition.get_balloon_at_frequency()`.
- [x] Database wrappers are split into dedicated modules exactly as shown in the early architecture sketch.

**Acceptance Criteria**

- [x] `from gll import GllFile` works after building/installing the local package.
- [x] Parsing from both a filesystem path and in-memory bytes returns structured Python objects.
- [x] Parsed JSON from the shared library is mapped into typed Python dataclasses and enums.
- [x] Resource, data-file, and include-file extraction are available for path-backed parses.
- [x] The implemented API matches the initial draft examples one-to-one without adaptation.

### 2.1 CFFI Interface (`python/gll/_ffi.py`)

```python
from cffi import FFI
import os

ffi = FFI()

ffi.cdef("""
    typedef struct {
        char* data;
        char* error;
        int64_t length;
    } GLL_Result;

    GLL_Result GLL_ParseFile(const char* path);
    GLL_Result GLL_ParseBytes(const char* data, int64_t length);
    GLL_Result GLL_ExtractResource(const char* path, int32_t index);
    GLL_Result GLL_ComputeArrayResponse(const char* config_json);
    void GLL_FreeResult(GLL_Result result);
    char* GLL_Version(void);
    void free(void* ptr);
""")

# Load the shared library
_lib_path = os.path.join(os.path.dirname(__file__), '_libgll.so')
lib = ffi.dlopen(_lib_path)


def _check_result(result) -> str:
    """Convert GLL_Result to Python string or raise exception."""
    try:
        if result.error != ffi.NULL:
            error_msg = ffi.string(result.error).decode('utf-8')
            raise GllError(error_msg)

        if result.data == ffi.NULL:
            return None

        return ffi.string(result.data).decode('utf-8')
    finally:
        lib.GLL_FreeResult(result)
```

### 2.2 Type Definitions (`python/gll/types.py`)

```python
from dataclasses import dataclass
from typing import List, Optional
from enum import Enum, auto


class SystemType(Enum):
    LINE_ARRAY = 0
    CLUSTER = 1
    LOUDSPEAKER = 2


@dataclass(frozen=True)
class Vector3D:
    x: float
    y: float
    z: float


@dataclass
class Metadata:
    product_name: str
    display_name: str
    manufacturer: str
    description: Optional[str] = None
    copyright: Optional[str] = None
    website: Optional[str] = None
    email: Optional[str] = None


@dataclass
class TransferFunction:
    frequencies: List[float]
    level: List[float]      # dB values
    phase: List[float]      # radians
    delay: float = 0.0


@dataclass
class BalloonData:
    """Directivity balloon at a single frequency."""
    frequency: float
    data: List[List[float]]  # [meridian][parallel] SPL values

    def get_spl(self, azimuth: float, elevation: float) -> float:
        """Get SPL at given angles (degrees)."""
        # Interpolation logic
        pass


@dataclass
class SourceDefinition:
    name: str
    index: int
    sensitivity_1w_1m: float
    impedance: float
    max_power: float
    balloon_data: List[BalloonData]
    frequency_response: Optional[TransferFunction] = None

    def get_balloon_at_frequency(self, freq: float) -> BalloonData:
        """Get balloon data interpolated to specific frequency."""
        pass


@dataclass
class BoxType:
    name: str
    index: int
    height: float
    width: float
    depth: float
    weight: float
    source_indices: List[int]


@dataclass
class Database:
    box_types: List[BoxType]
    source_definitions: List[SourceDefinition]
    frames: List['Frame']
    filter_groups: List['FilterGroup']
    # ... other fields
```

### 2.3 Main File Class (`python/gll/file.py`)

```python
import json
from pathlib import Path
from typing import Union, BinaryIO

from ._ffi import lib, ffi, _check_result
from .types import Metadata, Database, Resource
from .exceptions import GllError, ParseError


class GllFile:
    """Represents a parsed GLL file."""

    def __init__(self, data: dict, path: Optional[Path] = None):
        self._data = data
        self._path = path
        self._metadata = None
        self._database = None

    @classmethod
    def parse(cls, source: Union[str, Path, BinaryIO]) -> 'GllFile':
        """Parse a GLL file from path or file-like object."""
        if isinstance(source, (str, Path)):
            path = Path(source)
            result = lib.GLL_ParseFile(str(path).encode('utf-8'))
            json_str = _check_result(result)
            data = json.loads(json_str)
            return cls(data, path)
        else:
            # Read from file-like object
            content = source.read()
            result = lib.GLL_ParseBytes(content, len(content))
            json_str = _check_result(result)
            data = json.loads(json_str)
            return cls(data)

    @property
    def metadata(self) -> Metadata:
        """Get file metadata."""
        if self._metadata is None:
            m = self._data.get('metadata', {})
            self._metadata = Metadata(
                product_name=m.get('product_name', ''),
                display_name=m.get('display_name', ''),
                manufacturer=m.get('manufacturer', ''),
                description=m.get('description'),
                copyright=m.get('copyright'),
                website=m.get('website'),
                email=m.get('email'),
            )
        return self._metadata

    @property
    def database(self) -> Database:
        """Get parsed database with box types, sources, etc."""
        if self._database is None:
            self._database = Database.from_dict(self._data.get('database', {}))
        return self._database

    @property
    def resources(self) -> List[Resource]:
        """List embedded resources (images, PDFs, etc.)."""
        return [Resource.from_dict(r) for r in self._data.get('resources', [])]

    def extract_resource(self, resource: Resource) -> bytes:
        """Extract a resource's binary content."""
        if self._path is None:
            raise GllError("Cannot extract resources from in-memory parsed files")

        result = lib.GLL_ExtractResource(
            str(self._path).encode('utf-8'),
            resource.index
        )
        # Handle binary result
        ...
```

---

## Phase 3: Acoustic Calculations (2-3 weeks)

**Checklist**

- [x] `ArrayElement`, `ArrayConfig`, `AirProperties`, `ArrayResponse`, and `ArrayCalculator` exist in `python/gll/acoustics.py`.
- [x] The shared library exports `GLL_ComputeArrayResponse`.
- [x] Python callers can compute a combined transfer function for configured array elements.
- [x] Receiver position, air properties, and air attenuation options are exposed.
- [x] Convenience helpers exist for available box types, available sources, and response-grid evaluation.
- [ ] Combined balloon output is returned as part of `ArrayResponse`.
- [ ] Per-element transfer-function contributions are returned as part of `ArrayResponse`.
- [ ] The draft-only single-call `frequency` selector API is implemented exactly as shown in the original sketch.

**Acceptance Criteria**

- [x] A valid line-array configuration can return a transfer function through the Python API.
- [x] Invalid or empty configurations fail with Python exceptions instead of silent misuse.
- [x] Acoustic behavior is covered by automated Python tests using real GLL fixtures.
- [ ] The response payload contains every field from the original draft data model.

### 3.1 Array Configuration API

```python
from dataclasses import dataclass, field
from typing import List, Optional
import json

from .types import Vector3D, TransferFunction
from ._ffi import lib, _check_result


@dataclass
class ArrayElement:
    """Single element in an array configuration."""
    box_type: str
    position: Vector3D = field(default_factory=lambda: Vector3D(0, 0, 0))
    angles: Vector3D = field(default_factory=lambda: Vector3D(0, 0, 0))
    gain: float = 0.0
    delay: float = 0.0
    muted: bool = False


@dataclass
class ArrayConfig:
    """Line array configuration for response calculations."""
    elements: List[ArrayElement] = field(default_factory=list)

    def add_element(
        self,
        box_type: str,
        splay: float = 0.0,
        gain: float = 0.0,
        delay: float = 0.0,
    ) -> 'ArrayConfig':
        """Add an element to the array. Returns self for chaining."""
        # Calculate position based on previous elements
        element = ArrayElement(
            box_type=box_type,
            angles=Vector3D(0, splay, 0),
            gain=gain,
            delay=delay,
        )
        self.elements.append(element)
        return self

    def to_json(self) -> str:
        """Serialize to JSON for C API."""
        return json.dumps({
            'elements': [
                {
                    'box_type': e.box_type,
                    'position': {'x': e.position.x, 'y': e.position.y, 'z': e.position.z},
                    'angles': {'x': e.angles.x, 'y': e.angles.y, 'z': e.angles.z},
                    'gain': e.gain,
                    'delay': e.delay,
                    'muted': e.muted,
                }
                for e in self.elements
            ]
        })


@dataclass
class ArrayResponse:
    """Result of array response calculation."""
    transfer_function: TransferFunction
    combined_balloon: 'BalloonData'
    element_contributions: List[TransferFunction]

    @classmethod
    def from_json(cls, json_str: str) -> 'ArrayResponse':
        """Deserialize from C API response."""
        data = json.loads(json_str)
        # Convert to Python objects
        ...


class ArrayCalculator:
    """Compute array responses for a GLL file."""

    def __init__(self, gll_file: 'GllFile'):
        self.gll_file = gll_file

    def compute_response(
        self,
        config: ArrayConfig,
        frequency: Optional[float] = None,
        air_temperature: float = 20.0,
        air_humidity: float = 0.5,
    ) -> ArrayResponse:
        """
        Compute combined array response.

        Args:
            config: Array configuration
            frequency: Single frequency (Hz) or None for full spectrum
            air_temperature: Temperature in Celsius
            air_humidity: Relative humidity (0-1)

        Returns:
            ArrayResponse with combined transfer function and balloon
        """
        request = {
            'gll_path': str(self.gll_file._path),
            'config': json.loads(config.to_json()),
            'frequency': frequency,
            'air': {
                'temperature': air_temperature,
                'humidity': air_humidity,
            }
        }

        result = lib.GLL_ComputeArrayResponse(json.dumps(request).encode('utf-8'))
        json_str = _check_result(result)
        return ArrayResponse.from_json(json_str)
```

---

## Phase 4: NumPy Integration (2 weeks)

**Checklist**

- [x] `python/gll/numpy_compat.py` exists.
- [x] Optional NumPy dependency is declared in `python/pyproject.toml`.
- [x] Transfer-function and balloon conversion helpers are exposed for NumPy users.
- [x] NumPy helpers are exported from the package root and documented in `python/README.md`.
- [x] The shared library exports a raw float64 balloon-grid API for zero-copy access.
- [x] Python exposes a zero-copy `balloon_grid_view(...)` wrapper that keeps the C buffer alive for the ndarray lifetime.
- [x] Automated tests cover NumPy helper behavior.

**Acceptance Criteria**

- [x] Importing `gll` still works when NumPy is not installed.
- [x] NumPy helper entry points raise a clear `ImportError` when NumPy support is unavailable.
- [x] Copy-based conversion helpers return `float64` arrays with expected shapes and values.
- [x] Zero-copy balloon-grid access is available through the shared library and Python wrapper layer.

### 4.1 Optional NumPy Support

```python
# python/gll/numpy_compat.py
from typing import TYPE_CHECKING

try:
    import numpy as np
    HAS_NUMPY = True
except ImportError:
    HAS_NUMPY = False
    np = None

if TYPE_CHECKING:
    import numpy as np


def balloon_to_numpy(balloon: 'BalloonData') -> 'np.ndarray':
    """Convert balloon data to 2D numpy array."""
    if not HAS_NUMPY:
        raise ImportError("NumPy required for this feature. Install with: pip install gll[numpy]")
    return np.array(balloon.data, dtype=np.float64)


def transfer_function_to_numpy(tf: 'TransferFunction') -> tuple:
    """Convert transfer function to numpy arrays."""
    if not HAS_NUMPY:
        raise ImportError("NumPy required")
    return (
        np.array(tf.frequencies, dtype=np.float64),
        np.array(tf.level, dtype=np.float64),
        np.array(tf.phase, dtype=np.float64),
    )
```

### 4.2 Zero-Copy Option (Advanced)

For performance-critical applications, implement zero-copy array sharing:

```go
// In gll_cgo.go - return raw float64 arrays
//export GLL_GetBalloonRaw
func GLL_GetBalloonRaw(handle C.int64_t, freqIndex C.int32_t) C.GLL_RawArray {
    // Return pointer to float64 array with dimensions
    // Python can wrap with np.ctypeslib.as_array()
}
```

---

## Phase 5: Packaging & Distribution (2-3 weeks)

**Checklist**

- [x] `python/pyproject.toml` exists with project metadata, extras, and tool configuration.
- [x] The package declares optional `numpy` and `dev` dependency groups.
- [x] Local wheel builds are supported via `just build-python-wheel`.
- [x] Local editable install is supported via `just install-python`.
- [x] A dedicated GitHub Actions workflow builds Python wheels for Linux, macOS, and Windows.
- [ ] Tagged releases publish Python wheels to a package index.
- [ ] CI validates installation from built wheel artifacts.

**Acceptance Criteria**

- [x] A wheel can be built locally for the current platform.
- [x] The package metadata is sufficient for local editable installs, linting, typing, and pytest execution.
- [ ] Tagged releases automatically produce installable Python wheel artifacts for all supported platforms.
- [ ] `pip install gll` from a published package index works end-to-end on supported platforms.

### 5.1 pyproject.toml

```toml
[build-system]
requires = ["setuptools>=61", "wheel", "cffi>=1.0.0"]
build-backend = "setuptools.build_meta"

[project]
name = "gll"
version = "0.1.0"
description = "Python bindings for EASE GLL file parsing"
readme = "README.md"
license = {text = "MIT"}
requires-python = ">=3.10"
classifiers = [
    "Development Status :: 3 - Alpha",
    "Intended Audience :: Science/Research",
    "License :: OSI Approved :: MIT License",
    "Programming Language :: Python :: 3",
    "Programming Language :: Python :: 3.10",
    "Programming Language :: Python :: 3.11",
    "Programming Language :: Python :: 3.12",
    "Topic :: Multimedia :: Sound/Audio :: Analysis",
]
dependencies = [
    "cffi>=1.0.0",
]

[project.optional-dependencies]
numpy = ["numpy>=1.20"]
dev = [
    "pytest>=7.0",
    "pytest-cov",
    "mypy",
    "ruff",
]

[project.urls]
Homepage = "https://github.com/cwbudde/gll-tools"
Documentation = "https://gll-tools.readthedocs.io"
Repository = "https://github.com/cwbudde/gll-tools"
```

### 5.2 Platform Wheel Building

**GitHub Actions workflow:**

```yaml
name: Build Python Wheels

on:
  push:
    tags: ["v*"]

jobs:
  build-wheels:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
            wheel_plat: manylinux_2_17_x86_64
          - os: ubuntu-latest
            goos: linux
            goarch: arm64
            wheel_plat: manylinux_2_17_aarch64
          - os: macos-latest
            goos: darwin
            goarch: amd64
            wheel_plat: macosx_10_9_x86_64
          - os: macos-latest
            goos: darwin
            goarch: arm64
            wheel_plat: macosx_11_0_arm64
          - os: windows-latest
            goos: windows
            goarch: amd64
            wheel_plat: win_amd64

    runs-on: ${{ matrix.os }}

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"

      - uses: actions/setup-python@v5
        with:
          python-version: "3.11"

      - name: Build shared library
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: 1
        run: |
          go build -buildmode=c-shared -o python/gll/_libgll.so ./cmd/gllpy

      - name: Build wheel
        run: |
          pip install build wheel
          python -m build --wheel python/

      - name: Upload wheel
        uses: actions/upload-artifact@v4
        with:
          name: wheel-${{ matrix.wheel_plat }}
          path: python/dist/*.whl

  publish:
    needs: build-wheels
    runs-on: ubuntu-latest
    steps:
      - name: Download wheels
        uses: actions/download-artifact@v4

      - name: Publish to PyPI
        uses: pypa/gh-action-pypi-publish@release/v1
        with:
          packages-dir: wheel-*/
```

---

## Phase 6: Testing & Documentation (Ongoing)

**Checklist**

- [x] `python/tests/` covers parse, extract, acoustics, types, NumPy helpers, and zero-copy helpers.
- [x] The repo includes real GLL fixtures used by Python tests.
- [x] `pytest`, `mypy`, and `ruff` configuration exists in `python/pyproject.toml`.
- [x] `just` targets exist for Python test, lint, format, and typecheck flows.
- [x] User-facing Python documentation exists in `python/README.md` and is referenced from the root `README.md`.
- [ ] Automated memory-safety or leak-detection checks exist for the C/shared-library layer.
- [ ] The dedicated docs tree from the original draft (`docs/index.md`, `docs/api/`, `docs/guides/`, `docs/development/`) exists.
- [ ] Coverage targets are enforced or reported in CI for the Python bindings.

**Acceptance Criteria**

- [x] The Python package has an executable automated test suite that passes locally.
- [x] Static analysis and formatting tools are configured for the Python codebase.
- [x] A developer can find installation and usage guidance in the repository.
- [ ] Memory-safety validation for the shared library is automated.
- [ ] The structured multi-page documentation set proposed in the draft exists in the repository.

### 6.1 Test Strategy

| Level         | Tool                    | Coverage Target     |
| ------------- | ----------------------- | ------------------- |
| Unit tests    | pytest                  | 90% Python code     |
| Integration   | pytest + real GLL files | All public APIs     |
| Memory safety | valgrind + pytest       | No leaks in C layer |
| Type checking | mypy --strict           | 100% type coverage  |

### 6.2 Test Examples

```python
# tests/test_parse.py
import pytest
from pathlib import Path
from gll import GllFile


@pytest.fixture
def sample_gll():
    return Path(__file__).parent.parent.parent / "testdata" / "gll" / "sample.gll"


def test_parse_file(sample_gll):
    gll_file = GllFile.parse(sample_gll)
    assert gll_file.metadata.manufacturer != ""


def test_parse_bytes(sample_gll):
    with open(sample_gll, 'rb') as f:
        gll_file = GllFile.parse(f)
    assert gll_file.metadata is not None


def test_database_sources(sample_gll):
    gll_file = GllFile.parse(sample_gll)
    assert len(gll_file.database.source_definitions) > 0

    source = gll_file.database.source_definitions[0]
    assert source.sensitivity_1w_1m > 0


def test_balloon_interpolation(sample_gll):
    gll_file = GllFile.parse(sample_gll)
    source = gll_file.database.source_definitions[0]

    balloon = source.get_balloon_at_frequency(1000)
    on_axis = balloon.get_spl(0, 0)
    off_axis = balloon.get_spl(30, 0)

    # On-axis should be higher than 30° off-axis
    assert on_axis > off_axis
```

### 6.3 Documentation Structure

```
docs/
├── index.md
├── installation.md
├── quickstart.md
├── api/
│   ├── gll_file.md
│   ├── database.md
│   ├── acoustics.md
│   └── types.md
├── guides/
│   ├── extracting_resources.md
│   ├── array_calculations.md
│   └── numpy_integration.md
└── development/
    ├── building.md
    └── contributing.md
```

---

## Timeline Summary

| Phase                    | Duration  | Milestone                        |
| ------------------------ | --------- | -------------------------------- |
| 1. Foundation            | 4-6 weeks | Shared library builds, basic FFI |
| 2. Core Python API       | 3-4 weeks | `GllFile.parse()` works          |
| 3. Acoustic Calculations | 2-3 weeks | Array response API               |
| 4. NumPy Integration     | 2 weeks   | Zero-copy arrays                 |
| 5. Packaging             | 2-3 weeks | PyPI release                     |
| 6. Documentation         | Ongoing   | Full API docs                    |

**Total estimated time: 15-20 weeks**

---

## Risks & Mitigations

| Risk                            | Impact | Mitigation                                          |
| ------------------------------- | ------ | --------------------------------------------------- |
| Cross-compilation complexity    | High   | Use GitHub Actions with native runners per platform |
| Memory leaks in C layer         | Medium | Strict RAII patterns, valgrind testing              |
| JSON serialization overhead     | Medium | Profile hot paths, add binary protocol if needed    |
| cgo build issues on Windows     | Medium | Document MinGW setup, provide pre-built wheels      |
| API breakage during development | Low    | Semantic versioning, deprecation warnings           |

---

## Alternatives Considered

### Why not gopy?

- gopy generates bindings automatically but produces less idiomatic Python
- Manual approach allows better error handling and Pythonic API design
- More control over memory management and serialization

### Why not WASM + wasmtime-py?

- WASM has overhead for file I/O (must load entire file into memory)
- Less mature ecosystem for this use case
- cgo produces smaller, faster binaries

### Why not pure Python implementation?

- Would require reimplementing all parsing logic
- BitCompression decompression is complex
- Lose ability to share code with Go CLI tools

---

## Success Criteria

1. **Functionality**: Parse any GLL file that `gllinfo` can parse
2. **Performance**: <100ms parse time for typical files
3. **Memory**: No leaks, reasonable peak usage
4. **Usability**: Pythonic API, good error messages, type hints
5. **Distribution**: Single `pip install gll` works on Linux/macOS/Windows
