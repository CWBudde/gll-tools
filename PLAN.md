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

### 7.1 DXF Export (Cabinet Geometry)

- [x] Export CaseGeometry mesh data to DXF format
  - Convert GLL vertices/edges/faces to DXF ENTITIES (LINE, 3DFACE)
  - ASCII DXF R12 (AC1009) for maximum compatibility
  - RGB colors mapped to nearest ACI (AutoCAD Color Index)
  - Integrated into `gllinfo plot geometry --output file.dxf`
- [x] CLF `<CABINET>` tag references external DXF file path
  - `WithCabinetDXF(path)` export option writes `<CABINET>\t<dxf>path`
  - CLI flag `--cabinet-dxf` on `gllinfo acoustic --export-clf`
- [x] Add DXF export option to CLF export workflow (auto-generate DXF alongside CLF)
  - Auto-finds box geometry matching exported source
  - Generates `.dxf` file alongside `.clf` with same basename
  - References DXF basename in `<CABINET>` tag automatically

### 7.2 CLF Export (Common Loudspeaker Format)

- [x] Research CLF format specification
  - CLF text format is TAB-delimited with header fields and BAND data blocks
  - Two types: CLF1 (octave/10°) and CLF2 (1/3-octave/5°) — CLF2 matches GLL resolution
  - Binary format (CF1/CF2) is proprietary — targeting text format only
  - Full spec documented in [docs/clf-format.md](docs/clf-format.md)
- [x] Implement CLF text format writer (`internal/clf/`)
  - CLF2 text format with 5° resolution and 24 third-octave bands (100–20000 Hz)
  - Nearest-neighbor frequency resampling on log scale
  - GLL-to-CLF symmetry mapping (none/vertical/horizontal/full/rotational)
  - Full header metadata from GenSystem and SourceDefinition
- [x] Add CLI support for CLF export
  - `gllinfo acoustic --source N --export-clf output.txt`
- [x] Add tests for CLF export
  - Unit tests for frequencies, symmetry, writer, and export
  - Integration test: parse GLL → export CLF text → validate structure
- [x] Populate `<CABINET-SYSTEM>` text and `<WEIGHT>` from BoxType data
- [ ] Populate `<SENSITIVITY>` per-band data from on-axis response
- [ ] Handle `FrontHalfOnly` balloon data (front-hemisphere only export)

### 7.3 FRD Export (Frequency Response Data)

- [ ] Export frequency response to FRD format
  - Simple text format: `frequency magnitude phase`
  - One file per response/angle point
  - Include metadata in comments

### 7.4 Visualization (Optional)

- [x] Generate polar plots (SVG/PNG)
  - Plot level vs angle at specific frequencies
  - Support horizontal (meridian) and vertical (parallel) planes
  - Use go-chart or gonum/plot library
- [x] Generate frequency response graphs
  - Level and phase vs frequency
  - Support log frequency scale
- [x] Generate 3D balloon visualization
  - Export to common 3D formats (OBJ, PLY, STL)
  - Color vertices by SPL level
  - Use standard 72×37 grid (5° resolution)

### 7.5 XGLL Support (Text Format)

- [x] XGLL text format parser (`pkg/xgll/`)
  - Lexer/parser for all statement types (assignments, blocks, arrays)
  - `ParseFile()` reads .xgll into `Document` model
- [x] XGLL to GLL compilation — header + GenSystem
  - `BuildGLLFile()` maps XGLL Document → GLL File model
  - `gllEncoder` serializes header (v3–v6), checksums, hashes
  - GenSystem block: label, key, type, company, text fields, colors, flags
  - Round-trip via raw base64 blocks (`BinaryGenSystem`, `BinaryDatabase`, `BinaryTail`)
- [x] Writer registry with multiple output formats
  - `gll` (binary), `xgll` (text), `xgllbin` / `xgllbin-pretty` (JSON container)
  - CLI: `xgllc convert <file.xgll> -f gll -o out.gll`
- [x] Round-trip tests
  - `TestRoundTripXGLLSystem`: XGLL → GLL → XGLL GenSystem equivalence
  - `TestRoundTripGLLViaXGLL`: byte-for-byte GLL → XGLL → GLL (via raw blocks)
  - `TestGLLWriterRoundTripHeader`: header/GenSystem validation
- [x] Compiled GLL testdata from `testdata/xgll/*.xgll`
- [ ] Database serialization from XGLL (partial implementation)
  - [x] BoxTypes: basic metadata (label, key)
  - [x] BoxTypes: source placements (position, angles, source references)
  - [x] BoxTypes: physical properties (weight, reference points, opening angles)
  - [x] BoxTypes: CaseGeometry (3D mesh data - vertices, edges, faces)
    - Full binary encoding with Vertex/Edge/Face buffers
    - Supports symmetry, sub-versions, and all metadata fields
    - Round-trip tested with real GLL files
    - .xed export available via `gllinfo extract` and web demo
  - [ ] BoxTypes: InputConfigurations (inputs, links, rated impedance)
  - [ ] SourceDefinitions with balloon/transfer function data
  - [ ] FilterGroups with filter definitions
  - [ ] Limits, Warnings, Connectors, Frames

### 7.6 SOFA Export (Spatially Oriented Format for Acoustics)

- [ ] Export to SOFA format
  - HDF5-based format used in academic research
  - Requires: go-hdf5 library or h5py via cgo

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

---

**Remaining (lower priority):**

- Geometry parsing (for CaseGeometry in Frame) - complex, only needed for 3D visualization
- GenSystemConfig full parsing (currently only Label/Key extracted from presets)

---

## References

- `docs/research.md` - Comprehensive GLL format research
- `docs/format.md` - Binary format specification
- `docs/api.md` - Native DLL API documentation
- `testdata/` - Sample GLL files and XGLL examples
