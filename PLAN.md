# GLL Tools - Development Plan

## Project Goal

Develop Linux tools (in Go) to read and extract information from EASE GLL (Generic Loudspeaker Library) files without requiring Windows or the official EASE software.

## Project Structure

```text
gll-tools/
├── cmd/
│   └── gllinfo/          # CLI tool to inspect GLL files
├── pkg/
│   └── gll/              # Go library for parsing GLL files
├── docs/
│   ├── research.md       # Initial research document
│   └── format.md         # GLL format specification (reverse engineered)
├── testdata/             # Test data (GLL + XGLL examples)
├── legacy/
│   └── viewer/           # Extracted EASE GLL Viewer components
└── PLAN.md               # This file
```

---

## Phases 1–3: Setup & Research (Completed)

- [x] **Project Setup:** Initialized Go module, project structure, and test environment.
- [x] **Reverse Engineering:** Analyzed GLL Viewer and related documentations found on the internet.
- [x] **Documentation:** Documented binary format in [docs/format.md](docs/format.md) and native APIs in [docs/api.md](docs/api.md).

---

## Phase 4: Go Library - Core Parser (`pkg/gll`) ✓

### 4.1–4.2 Header & Metadata ✓

EGLL magic, version parsing, checksum/hash, and all GenSystem metadata fields.

### 4.3 Embedded Resources ✓

PNG images, zlib-compressed blocks, PDF/fonts/text, and 3D geometry (.xed) via DataFiles.

### 4.4 Acoustic Data ✓

SourceDefinitions with BalloonData, TransferFunctions (CLogSpectrumLP v0 and TransferFunctionLsPs v1),
BitCompression decompression with variable bit-depth and delta encoding.

### 4.5 Advanced Structures ✓

All database buffers implemented:

- BoxTypes, FilterGroups, Limits, Warnings
- Frames (vcheck=1, with PinPoints), Connectors (vcheck=1, with Angles)
- BoxInputConfig (source-filter links)
- ClusterSetups (predefined arrays), Presets (sver≥1), IncludeFiles/AuthorFiles (sver≥2), Transformers (sver≥3)

**Note:** Geometry parsing skipped (only needed for 3D visualization).

- CaseGeometry field in Frame is skipped using block size to seek past it

---

## Phase 5: CLI Tool (`cmd/gllinfo`)

### 5.1 Basic Functionality

- [x] Read GLL file argument
- [x] Display header information
- [x] Display metadata (manufacturer, model, description)
- [x] List embedded resources

### 5.2 Extended Functionality

- [x] `--extract-images` - extract embedded PNG files (via `extract --images`)
- [x] `--extract-all` - extract all embedded resources (via `extract` subcommand)
- [x] `--json` - output in JSON format
- [x] `--verbose` - detailed structure dump

### 5.3 Acoustic Data Display

- [x] `acoustic` subcommand - display acoustic sources and balloon data
- [x] `--source N` - display specific source in detail
- [x] `--responses` - load and display frequency/phase response data
- [x] `--max-responses N` - limit number of responses displayed
- [x] `--export-csv` - export response data to CSV format
  - Columns: response_index, meridian_deg, parallel_deg, frequency_hz, level_db, phase_rad

### 5.4 Configuration Display ✓

- [x] `config` subcommand - display limits, warnings, filter groups
- [x] `--limits` - show mechanical/electrical limits
- [x] `--warnings` - show configuration warnings
- [x] `--filters` - show filter group definitions
- [x] `--json` - output in JSON format

---

## Phase 6: Validation & Testing ✓

### 6.1 Test Suite

- [x] Unit tests for header parsing (`pkg/gll/parse_test.go`)
  - Test EGLL magic detection
  - Test version parsing (v3-v6)
  - Test checksum validation (v4+)
  - Test invalid magic/format handling
- [x] Unit tests for types (`pkg/gll/types_test.go`)
  - Test SystemType, DataType, SymmetryType strings
  - Test ResolutionDescriptor calculations
  - Test LogSpectrumDefinition frequency calculations
- [x] Unit tests for database structures (`pkg/gll/database_test.go`)
  - Test LimitType/WarningType strings
  - Test struct field assignments
- [x] Unit tests for acoustic data
  - Test BitCompression decompression (`internal/gll/bitcompression_test.go`)
  - Test ResolutionDescriptor parsing (`pkg/gll/source_test.go`)
  - Test record parsing (uncompressed/compressed)
- [x] Integration tests with real GLL files (`pkg/gll/integration_test.go`)
  - Test with testdata/gll/\*.gll (3 files)
  - Test with legacy/viewer/\*.gll (10 files)
  - Test database parsing, source definitions, resource extraction

### 6.2 Cross-Validation

- [ ] Compare extracted data with GLL Viewer output
  - Export from viewer, compare JSON output
- [ ] Verify directivity values match
  - Compare balloon grid dimensions
  - Verify response count matches
- [ ] Verify frequency response data matches
  - Compare level/phase arrays at key frequencies
  - Verify scale factors (0.01 dB, 0.001 rad)
- [ ] Verify limits and warnings match
  - Compare limit types and values
  - Compare warning messages
- [ ] Document any discrepancies in `docs/validation.md`

---

## Phase 7: Advanced Features (Future)

### 7.1 Data Export

- [ ] Export directivity to CLF format (Common Loudspeaker Format)
  - CLF is industry-standard for acoustic simulation software
  - Map balloon data to CLF structure (header + data blocks)
  - Reference: <https://www.clfgroup.org/>
- [ ] Export frequency response to FRD format
  - Simple text format: `frequency magnitude phase`
  - One file per response/angle point
  - Include metadata in comments
- [ ] Export to SOFA format (Spatially Oriented Format for Acoustics)
  - HDF5-based format used in academic research
  - Requires: go-hdf5 library or h5py via cgo

### 7.2 Visualization (Optional)

- [ ] Generate polar plots (SVG/PNG)
  - Plot level vs angle at specific frequencies
  - Support horizontal (meridian) and vertical (parallel) planes
  - Use go-chart or gonum/plot library
- [ ] Generate frequency response graphs
  - Level and phase vs frequency
  - Support log frequency scale
- [ ] Generate 3D balloon visualization
  - Export to common 3D formats (OBJ, PLY, STL)
  - Color vertices by SPL level
  - Use standard 72×37 grid (5° resolution)

### 7.3 Wine Integration (Optional)

- [ ] Call native DLL functions via Wine
  - Use CGO with Wine's PE loader or go-ole
  - Validate parsed data against official API output
- [ ] Document native API usage patterns
  - See `docs/api.md` for function signatures
  - Buffer sizes: DB_GetCone5=108, DB_GetLobe5=1332, DB_GetPhase5=2664 floats
  - Frequency bands: 21 (1/3-octave, 50Hz-10kHz)
- [ ] Hybrid approach for complex calculations
  - Use native DLL for proprietary interpolation algorithms
  - Use Go parser for file I/O and data export

### 7.4 XGLL Support (Text Format)

- [ ] Complete XGLL text format parser
  - Existing: `pkg/xgll/` with basic lexer/parser
  - Document block structure and statement syntax
  - Parse all statement types (assignments, blocks, arrays)
- [ ] XGLL to GLL compilation (write support)
  - Implement binary serialization (SaveToTarget pattern)
  - Generate valid checksums (v4+) and hashes (v6)
  - Validate output against reference files

## Phase 8: WebAssembly Demo

### 8.1 Basic WebAssembly Demo

- [x] Create WASM entry point (`cmd/gllwasm/main.go`)
  - Exports `parseGLL(Uint8Array) → JSON` function
  - Returns header, metadata, sources, and responses
- [x] Create web interface (`web/`)
  - Drag-and-drop file upload
  - Tabbed interface: Overview, Sources, Responses, Config, Resources
  - Frequency response charts with Chart.js
  - Data tables with frequency/level/phase values
- [x] GitHub Actions deployment (`.github/workflows/pages.yml`)
  - Auto-deploy to GitHub Pages on push to main
  - Builds WASM from Go source

### 8.2 Advanced Features (Future)

- [x] Add polar plot visualization
  - 2D polar plots at selected frequencies
  - Horizontal and vertical plane views
- [x] Add 3D balloon visualization
  - Interactive 3D directivity balloon
  - Color-coded by SPL level
  - Requires Three.js or similar
- [ ] Add data export options
  - Export response data as CSV
  - Export directivity as CLF or FRD

---

## Current Status

Phases 4-6 are complete. Core parser, CLI tool, and test suite are implemented.

### Research Phase Complete (Phase 1-3)

- Project structure created
- Comprehensive format documentation in `docs/format.md`
- Internal data structures:
  - Component structure (position, angles, filters)
  - BandData structure (directivity, sensitivity, impedance per band)
  - Buffer sizes: DB_GetCone5=108, DB_GetLobe5=1332, DB_GetPhase5=2664 floats
  - Plugin entry: 0x4230 bytes with function pointers at +0x4224/+0x4228/+0x422C
- Calling conventions: stdcall, global state management
- Frequency bands: 21 (1/3-octave, 50Hz-10kHz)
- Directivity grid: 72×37 points (5° resolution)

### Go Parser Status (Phase 4 nearly complete)

**Completed:**

- Header parsing (magic, version, checksum, hash)
- GenSystem metadata extraction (all fields)
- Database buffer parsing:
  - DataFiles (embedded PNGs, XED geometry files)
  - BoxTypes (cabinet definitions with Label, Key)
  - Frames (rigging frames with PinPoints, NextPivot, CenterOfMass)
  - Connectors (box-to-box connections with splay angles)
  - Limits (mechanical/electrical constraints)
  - Warnings (configuration messages)
  - FilterGroups with nested FilterDefinitions
  - BoxInputConfig (parsing ready, not yet integrated into BoxType)
- SourceDefinitions with BalloonData:
  - Angular resolution (symmetry, steps, grid size)
  - Response metadata (count, version, offset)
- Transfer function parsing:
  - CLogSpectrumLP (v0 legacy format)
  - TransferFunctionLsPs (v1 current format)
  - ComplexSequence with nested Records
- BitCompression decompression algorithm (`internal/gll/bitcompression.go`)
- CLI tool (`cmd/gllinfo`) with subcommands:
  - Default: header + metadata display
  - `extract`: resource extraction (images, data)
  - `acoustic`: source definitions with response loading

**Files added/modified:**

- `pkg/gll/types.go` - Core type definitions
- `pkg/gll/database.go` - Database buffer parsing:
  - Limits, Warnings, FilterGroups
  - Frames (with LabeledVector3DBuffer for PinPoints)
  - Connectors (with LabeledValueDBuffer for Angles)
  - BoxInputConfig (with BoxInput and SourceFilterLink)
  - ClusterSetups (with ClusterSetupItem, ClusterSetup, ClusterBox)
  - Presets (GenSystemPreset with Label/Key)
  - IncludeFiles (with Label, Key, Filename, embedded bytes)
  - AuthorFiles (uses DataFileBuffer format)
  - Transformers (with TapSettingBuffer)
- `pkg/gll/source.go` - Source definitions and transfer functions
- `internal/gll/bitcompression.go` - Decompression algorithm
- `cmd/gllinfo/cmd/acoustic.go` - Acoustic data display
- `docs/format.md` - Format specification (updated with new structures)

**Remaining (lower priority):**

- Geometry parsing (for CaseGeometry in Frame) - complex, only needed for 3D visualization
- GenSystemConfig full parsing (currently only Label/Key extracted from presets)

---

## References

- `docs/research.md` - Comprehensive GLL format research
- `docs/format.md` - Binary format specification
- `docs/api.md` - Native DLL API documentation
- `testdata/` - Sample GLL files and XGLL examples
