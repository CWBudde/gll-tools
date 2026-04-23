# GLL Tools

Go library and CLI tools for reading EASE GLL (Generic Loudspeaker Library) files on Linux.

**[Try it online](https://cwbudde.github.io/gll-tools/)** - WebAssembly demo running in your browser

## Overview

GLL files are a proprietary format used by AFMG's EASE acoustic simulation software to describe loudspeaker systems. This project aims to reverse-engineer the format and provide open-source tools for extracting information from these files.

## Features

- Parse GLL file headers and metadata
- Extract manufacturer/product information
- Extract embedded resources (images, fonts, PDFs)
- Decompress zlib-compressed resources
- Parse database structures (box types, source definitions, filter groups, limits, warnings, presets)
- Decode acoustic directivity balloon data and frequency responses
- Export response data to CSV
- Export plots (polar, response) and 3D balloon meshes
- JSON output format
- Convert between GLL binary and XGLL text formats

## Installation

```bash
go install github.com/cwbudde/gll-tools/cmd/gllinfo@latest
go install github.com/cwbudde/gll-tools/cmd/xgllc@latest
```

## Web Demo Smoke Test

The web demo has one minimal Playwright smoke test. It checks that the browser app loads, parses a sample `.gll`, and renders the visualization tab's expected state for the sample fixture.

```bash
just build-wasm
npm install
npm run test:web
```

## Usage

### Display file information

```bash
# Basic information
gllinfo speaker.gll

# Verbose output (includes full descriptions, balloon details, checksums)
gllinfo -v speaker.gll

# JSON output
gllinfo --json speaker.gll
```

### Extract embedded resources

```bash
# Extract all resources to current directory
gllinfo extract speaker.gll

# Extract to specific directory
gllinfo extract --output ./extracted speaker.gll

# Extract only images (PNG files)
gllinfo extract --images speaker.gll

# Extract only data files (.xed geometry files)
gllinfo extract --data speaker.gll

# Extract without decompressing zlib resources
gllinfo extract --decompress=false speaker.gll
```

### Acoustic data

```bash
# Show all source definitions with balloon info
gllinfo acoustic speaker.gll

# Show a specific source in detail
gllinfo acoustic speaker.gll --source 0

# Include frequency response data
gllinfo acoustic speaker.gll -s 0 --responses

# Export response data to CSV
gllinfo acoustic speaker.gll -s 0 --export-csv output.csv
```

### Visualization exports

```bash
# Polar directivity plot (SVG)
gllinfo plot polar speaker.gll --source 0 --frequency 1000 --output polar.svg

# Frequency response plot (SVG)
gllinfo plot response speaker.gll --source 0 --mode magnitude --output response.svg
gllinfo plot response speaker.gll --source 0 --mode phase-wrapped --output response_phase_wrapped.svg
gllinfo plot response speaker.gll --source 0 --mode phase-unwrapped --output response_phase_unwrapped.svg
gllinfo plot response speaker.gll --source 0 --mode group-delay --output response_group_delay.svg

# 3D balloon mesh (STL/OBJ)
gllinfo plot balloon speaker.gll --source 0 --frequency 1000 --output balloon.stl
gllinfo plot balloon speaker.gll --source 0 --frequency 1000 --output balloon.obj
gllinfo plot balloon speaker.gll --source 0 --frequency 1000 --output balloon_centered.obj --center bbox

# Cabinet/frame geometry export (STL/OBJ)
gllinfo plot geometry speaker.gll --box 0 --output box_geometry.obj
gllinfo plot geometry speaker.gll --frame 0 --output frame_geometry.stl
```

### Configuration data

```bash
# Show all config (limits, warnings, filter groups)
gllinfo config speaker.gll

# Show only limits, warnings, or filters
gllinfo config speaker.gll --limits
gllinfo config speaker.gll --warnings
gllinfo config speaker.gll --filters
```

### System presets

```bash
# Show presets
gllinfo presets speaker.gll

# Decode config bytes
gllinfo presets speaker.gll --decode
```

### Convert GLL to XGLL text

```bash
# Convert to XGLL and print to stdout
gllinfo xgll speaker.gll

# Write to file
gllinfo xgll speaker.gll -o output.xgll
```

### Parse XGLL files

```bash
# Parse and show diagnostics
xgllc parse testdata/xgll/example-ls.xgll

# JSON output
xgllc --json parse testdata/xgll/example-la.xgll

# Validate system-specific blocks
xgllc validate testdata/xgll/example-la.xgll

# Convert to a binary container
xgllc convert testdata/xgll/example-ls.xgll --output /tmp/example.xgllbin --format xgllbin

# Convert with a pretty JSON payload
xgllc convert testdata/xgll/example-ls.xgll --output /tmp/example-pretty.xgllbin --format xgllbin --pretty

# Convert to a minimal GLL container (header + GenSystem only)
xgllc convert testdata/xgll/example-ls.xgll --output /tmp/example.gll --format gll

# Convert a GLL file back to XGLL text
xgllc from-gll speaker.gll -o output.xgll
```

### Other commands

```bash
# Inspect trailing bytes after the GenSystem block
gllinfo tail speaker.gll

# Show version information
gllinfo version

# Show help
gllinfo --help
```

## Python Bindings

Native Python bindings are available via a shared library built with cgo:

```bash
# Build and install
just build-python
pip install -e ./python

# Build release-style shared libraries for common targets
just build-python-all

# Or use pip directly after building
CGO_ENABLED=1 go build -buildmode=c-shared -o python/gll/_libgll.so ./cmd/gllpy
pip install ./python
```

```python
from gll import GllFile, ArrayCalculator, ArrayConfig

# Parse a GLL file
gll = GllFile.parse("speaker.gll")
print(f"Manufacturer: {gll.metadata.manufacturer}")
print(f"Product: {gll.metadata.product_name}")

# Extract resources
for res in gll.resources:
    data = gll.extract_resource(res)
    with open(res.name, 'wb') as f:
        f.write(data)

# Compute array response
calc = ArrayCalculator(gll)
config = ArrayConfig().add_element("K2", splay=0.5).add_element("K2", splay=1.0)
response = calc.compute_response(config)
```

See [python/README.md](python/README.md) for full documentation.

## C API (for LabVIEW, MATLAB, C#, etc.)

The shared library (`libgll.so` / `libgll.dll`) provides a C API that can be used from any language supporting C FFI:

```bash
# Build the shared library
CGO_ENABLED=1 go build -buildmode=c-shared -o libgll.so ./cmd/gllpy
```

**Supported platforms:**

- Linux (x86_64, arm64)
- macOS (x86_64, arm64)
- Windows (x86_64, arm64) - requires MinGW for building

**API Functions:**

| Function                                     | Description                  |
| -------------------------------------------- | ---------------------------- |
| `GLL_ParseFile(path)`                        | Parse GLL file, returns JSON |
| `GLL_ParseBytes(data, len)`                  | Parse GLL from memory        |
| `GLL_ExtractResource(path, idx)`             | Extract embedded resource    |
| `GLL_ExtractDataFile(path, idx)`             | Extract data file            |
| `GLL_ComputeArrayResponse(json)`             | Compute array response       |
| `GLL_GetBalloonAtFrequency(path, src, freq)` | Get directivity data         |
| `GLL_FreeResult(result)`                     | Free returned memory         |

See [docs/c-api.md](docs/c-api.md) for detailed C API documentation including LabVIEW examples.

## Project Structure

```text
gll-tools/
├── cmd/
│   ├── gllinfo/      # CLI for inspecting/extracting GLL files
│   ├── xgllc/        # CLI for parsing/converting XGLL text format
│   ├── gllwasm/      # WebAssembly build for browser-based parsing
│   └── gllpy/        # C shared library for Python/LabVIEW/etc.
├── pkg/
│   ├── gll/          # Public library for GLL binary parsing
│   └── xgll/         # Public library for XGLL text format parsing
├── python/           # Python bindings package
├── internal/
│   ├── gll/          # Internal parsing utilities (ByteReader, BitCompression)
│   ├── acoustics/    # Acoustic calculations (air properties, balloon geometry)
│   ├── compression/  # BitCompression decompression
│   ├── filters/      # IIR filter processing
│   └── mime/         # MIME type detection for embedded resources
├── docs/             # Format specifications and research
├── testdata/         # Test GLL and XGLL files
└── legacy/           # Extracted EASE GLL Viewer (for research)
```

## What Can Be Extracted

The `gllinfo` tool can extract:

- **Metadata**: Manufacturer, model name, description, copyright, website, email, support info
- **Box Types**: Cabinet configurations and cluster setups
- **Source Definitions**: Driver specifications with frequency ranges, directivity balloon data, and frequency/phase responses
- **Acoustic Data**: Balloon directivity grids with symmetry information, individual frequency responses exportable to CSV
- **Configuration**: Mechanical/electrical limits, warnings, and filter group definitions (Butterworth, Bessel, Linkwitz-Riley, etc.)
- **Presets**: System preset configurations
- **Images**: Embedded PNG images (product photos, diagrams)
- **Geometry**: 3D geometry file references (.xed files)
- **Compressed Resources**: Zlib-compressed embedded data (PDFs, fonts, text)

## Example Output

```text
File: speaker.gll
Format: EGLL v6 (sub: 0)

=== System ===
Label:   AcmeArray
Key:     AcmeArray
Type:    LineArray
Version: 2.1

=== Metadata ===
Manufacturer: Acme Acoustics GmbH
Description:  AcmeArray is a compact line array element...
Copyright:    © 2024 Acme Acoustics
Website:      http://www.acme-acoustics.example.com
Email:        contact@acme-acoustics.example.com

=== Box Types ===
  AcmeArray 60°
  AcmeArray 90°

=== Source Definitions ===
  LF: 50-700 Hz (HighRes)
  MHF: 700-20000 Hz (HighRes)

=== Embedded Resources ===
  PNG: .\Drawings\AcmeArray.png (11796 bytes)
```

## Documentation

- [docs/format.md](docs/format.md) - GLL file format specification
- [docs/research.md](docs/research.md) - Background research
- [docs/api.md](docs/api.md) - Go API documentation
- [docs/c-api.md](docs/c-api.md) - C API documentation (for LabVIEW, MATLAB, C#, etc.)
- [python/README.md](python/README.md) - Python bindings documentation

## Building from Source

```bash
# Clone the repository
git clone https://github.com/cwbudde/gll-tools.git
cd gll-tools

# Build all CLI tools
just build

# Or build individually
go build -o gllinfo ./cmd/gllinfo
go build -o xgllc ./cmd/xgllc

# Run tests
just test

# Format and lint
just fmt
just lint

# Run all checks
just check
```

## Legal Context

This project is for educational and interoperability purposes. GLL is a proprietary format owned by AFMG. The project does not redistribute any AFMG software or copyrighted material - only reverse-engineered format specifications and clean-room implementations.

## License

MIT License - See LICENSE file for details.
