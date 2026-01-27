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
- Parse database structures (box types, source definitions)
- JSON output format
- Extract directivity data
- Extract frequency response data
- Export to open formats (CLF, FRD, CSV)

## Installation

```bash
go install github.com/cwbudde/gll-tools/cmd/gllinfo@latest
go install github.com/cwbudde/gll-tools/cmd/xgllc@latest
```

## Usage

### Display file information

```bash
# Basic information
gllinfo speaker.gll

# Verbose output (includes full descriptions and checksums)
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

### Other commands

```bash
# Show version information
gllinfo version

# Show help
gllinfo --help
gllinfo extract --help
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
```

## Project Structure

```
gll-tools/
├── cmd/gllinfo/     # CLI tool
├── cmd/xgllc/       # XGLL parser/converter CLI tool
├── pkg/gll/         # Go library for parsing GLL files
├── pkg/xgll/        # Go library for parsing XGLL files
├── docs/            # Documentation and format specifications
├── testdata/        # Test data (GLL + XGLL examples)
└── legacy/          # Extracted EASE GLL Viewer (for research)
```

## What Can Be Extracted

The `gllinfo` tool can currently extract:

- **Metadata**: Manufacturer, model name, description, copyright, website, email, support info
- **Images**: Embedded PNG images (product photos, diagrams)
- **Geometry**: 3D geometry file references (.xed files)
- **Compressed Resources**: Zlib-compressed embedded data including:
  - PDF content (graphics and font data)
  - TrueType fonts (TTF files)
  - Text data
- **Database Information**:
  - Box types (cabinet configurations)
  - Source definitions (driver specifications with frequency ranges)
  - Data files (embedded resources)

## Example Output

```
File: TiRAY-V1_3.gll
Format: EASE v3 (sub: 3)

=== System ===
Label:   TiRAY
Key:     TiRAY
Type:    Line Array
Version: 1.3

=== Metadata ===
Manufacturer: d&b audiotechnik GmbH
Description:  d&b TiRAY Line Array Element...
Website:      www.dbaudio.com

=== Source Definitions ===
  Woofer: 60-200 Hz (Monopole Radial Symmetric)
  MF Driver: 200-2000 Hz (Rotating balloon)
  HF Driver: 2000-20000 Hz (Rotating balloon)

=== Embedded Resources ===
  png: TiRAY.png (245678 bytes)
  zlib: pdf-graphics (12345 bytes)
  zlib: font-ttf (45678 bytes)
```

## Documentation

- [docs/format.md](docs/format.md) - GLL file format specification
- [docs/research.md](docs/research.md) - Background research
- [docs/api.md](docs/api.md) - API documentation

## Building from Source

```bash
# Clone the repository
git clone https://github.com/cwbudde/gll-tools.git
cd gll-tools

# Build the CLI tool
go build -o gllinfo ./cmd/gllinfo

# Run tests
go test ./...

# Format and lint (requires justfile)
just fmt
just lint
```

## Legal Context

This project is for educational and interoperability purposes. GLL is a proprietary format owned by AFMG. The project does not redistribute any AFMG software or copyrighted material - only reverse-engineered format specifications and clean-room implementations.

## License

MIT License - See LICENSE file for details.
