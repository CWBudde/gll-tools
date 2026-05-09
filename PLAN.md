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
  - [x] BoxTypes: InputConfigurations (inputs, links, rated impedance)
    - XGLL text output via `buildInputConfigStatements()`
    - XGLL text parsing via `parseInputConfigStatement()`
    - Round-trip tested: GLL → XGLL → GLL preserves InputConfig data
    - Synthetic encoder round-trip covers InputConfig + CaseGeometry + opening
      angles via `TestEncoderSynthetic_BoxTypeFull` (encoder now wraps the
      InputConfigBuffer in its outer `int32` size header to match
      `parseInputConfigBuffer`)
  - [x] SourceDefinitions with balloon/transfer function data
    - XGLL text output via `buildSourceDefinitionStatements()` emits
      human-readable metadata (Label, Bandwidth, DataType, OnAxisLevel,
      Company, Description) plus a `BinarySourceDefinition` base64 blob
      that preserves BalloonData and OnAxisSpectrum
    - XGLL text parsing via `parseSourceDefinitionStatements()` inflates the
      blob using the new exported `gll.ParseSourceDefinitionItemBytes` helper,
      which eagerly loads balloon responses for self-contained items
    - Round-trip tested: `TestRoundTripSourceDefinitionsViaXGLLText` covers
      GLL → XGLL text → GLL → binary GLL → re-parse with metadata + balloon
      response counts preserved
  - [x] FilterGroups with filter definitions
    - XGLL text output via `buildFilterGroupStatements()` emits human-readable
      metadata (Label, Key, IsOverridable, FilterDefinition labels/keys) plus
      a `BinaryFilterGroup` base64 blob that preserves the full filter bank
      data (IIR/FIR/LogSpectrum) without reimplementing the FilterGroup
      binary encoder
    - Raw FilterGroup bytes are now captured during `parseFilterGroup` into a
      new `FilterGroup.RawBlock` field and re-inflated via the exported
      `gll.ParseFilterGroupBytes` helper
    - XGLL text parsing via `parseFilterGroupStatements()` skips loose
      `FilterDefinition` lines whenever a `BinaryFilterGroup` blob has
      already populated the filters (blob is the source of truth)
    - Round-trip tested: `TestRoundTripFilterGroupsViaXGLLText` loads
      `testdata/gll/3Way-LR.gll` (9 FilterGroups), runs GLL → XGLL text →
      GLL, and verifies each group's Label/Key/IsOverridable plus per-filter
      Label/Key are preserved
  - [x] Limits, Warnings, Connectors, Frames
    - Each type captures `RawBlock []byte` during its binary parser
      (`parseLimit`, `parseWarning`, `parseConnector`, `parseFrame`) so the
      original on-disk bytes can be re-emitted verbatim
    - XGLL text output via `buildLimitStatements`, `buildWarningStatements`,
      `buildConnectorStatements`, `buildFrameStatements` emits the
      human-readable metadata (Frame/Type/BoxType/Value, Angles list,
      PinPoints, etc.) plus a `BinaryLimit` / `BinaryWarning` /
      `BinaryConnector` / `BinaryFrame` base64 blob
    - XGLL text parsing inflates each blob via the new exported
      `gll.ParseLimitBytes` / `gll.ParseWarningBytes` /
      `gll.ParseConnectorBytes` / `gll.ParseFrameBytes` helpers; the blob is
      the source of truth and metadata-only fallback is kept lenient for
      hand-edited XGLL fixtures
    - Round-trip tested: `TestRoundTripLimitsWarningsViaXGLLText`,
      `TestRoundTripConnectorsViaXGLLText`,
      `TestRoundTripFramesViaXGLLText` use `testdata/gll/APS-V1_1.gll`
      (L=11, W=2, C=23, F=3) and verify counts + per-item field equality
      after GLL → XGLL text → GLL

### 7.6 SOFA Export (Spatially Oriented Format for Acoustics)

- [x] TF export via FreeFieldDirectivityTF (one .sofa per BalloonData)
  - go-sofa extended upstream with TF read+write support (Frequencies,
    TFReal, TFImag fields; Save/Open dispatch on DataType)
  - `pkg/sofaexport` library: BuildSOFAFile (pure), ExportSourceBalloon,
    ExportFile (parses GLL and emits one .sofa per source/balloon)
  - GLL-native log frequency grid preserved (no resampling)
  - Combined absolute TF by default (balloon × OnAxisSpectrum, scaled by
    OnAxisLevel); `--relative` flag emits raw balloon TF
  - CLI: `gllinfo export sofa <file.gll> [-o dir] [--relative] [--source]
[--use-case] [--pattern] [--overwrite] [-v]`
  - Round-trip tests pass against `testdata/gll/3Way-LR.gll`
- [ ] FIR export via IFFT (`--fir` flag, GeneralFIR convention)
- [ ] External validation against SOFA Toolbox / pysofaconventions

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

## Phase 9: Visualization Tab Validation & Refinement

Goal: make the web demo visualization tab trustworthy, polished, and easy to compare against known reference output. The parser and non-visual tabs are considered mostly stable; this phase focuses on array response visualization, acoustic assumptions in the visualization path, and UI quality.

### 9.1 Acoustic Contract Review

- [ ] Document the exact contract for `BalloonData.Responses`:
  - Are balloon responses relative directivity only, absolute SPL, or already combined with on-axis data?
  - Confirm whether visualization and array calculations should combine `SourceDefinition.OnAxisSpectrum` or the balloon response at `(meridian=0, parallel=0)`.
  - [x] Add initial notes to `docs/format.md` and `docs/acoustic-model.md`.
- [ ] Validate propagation delay phase convention:
  - Confirm whether stored phase uses the same sign convention as Go's complex summation.
  - [x] Add a synthetic two-source interference test where expected nulls/peaks are analytically known.
  - Align `TransferFunction.AddDelay`, web phase display, and group-delay calculation semantics.
- [ ] Validate angular interpolation:
  - [x] Replace direct linear interpolation of wrapped phase with unit-circle interpolation.
  - [x] Replace array-calculation angular interpolation with full complex-pressure interpolation.
  - [x] Decide whether display-only visualization paths should use dB interpolation or the same complex-pressure interpolation.
  - [x] Add tests around phase wrap boundaries near `+π/-π`.
- [ ] Validate air attenuation expectations:
  - [x] Remove approximate UI/docs wording after replacing the simplified model.
  - [x] Replace the simplified attenuation model with ISO 9613-1 atmospheric absorption.
  - [x] Add ISO 9613-1 reference-value tests across frequency, humidity, temperature, and pressure.
  - [x] Wire temperature/pressure inputs consistently through Go, WASM, web, and Python APIs.
  - [ ] Cross-check ISO 9613-1 coefficients against an external calculator or published reference table before treating the web demo as prediction-grade.

### 9.2 Coordinate & Placement Consistency

- [x] Audit coordinate conventions across Go, WASM, Python, and the web UI:
  - Confirm that `+X` is the firing/on-axis direction everywhere.
  - Fix or document legacy/default paths that still treat `+Y` as front.
  - Add tests for receiver placement at front/right/top/back.
- [x] Validate box/source placement handling:
  - Confirm that every source placement position and H/V/R angle is applied before array summation.
  - Ensure web, WASM, CLI/Python bindings, and any future API use the same expansion model.
  - Add a multi-way synthetic box test where two source offsets produce a predictable phase difference.
- [x] Review rotation matrix composition:
  - Verify `buildRotationMatrix()` against GLL H/V/R conventions and the Go fallback Euler path.
  - Add cross-language tests for several known rotations and direction-to-GLL-angle mappings.

### 9.3 Reference Validation

- [x] Create a reference comparison workflow:
  - Export screenshots/data from official EASE GLL Viewer for one small loudspeaker and one line-array sample.
  - Compare on-axis response, off-axis response, polar slices, and combined array response.
  - Store tolerances and discrepancy notes in `docs/validation.md`.
- [ ] Add deterministic web demo fixtures:
  - [x] Include at least one tiny synthetic fixture with known directivity and phase behavior for browser-facing computation code.
  - Extend Playwright smoke tests to check plotted metadata and selected numeric chart values, not just page visibility.
- [ ] Add regression tests for visualization-facing WASM APIs:
  - [x] `computeArrayResponse`
  - [x] `computeArrayBalloon`
  - [x] `computeArrayBalloonAsync` progress callback and final result parity at the shared computation-core level
  - [ ] Browser-level JS wrapper parity test after server/browser-based tests are allowed again

### 9.4 Visualization UX Sophistication

- [x] Show determinate progress while computing array balloon grids.
- [x] Make visual computation state clearer:
  - Show cached vs stale state when configuration changes and auto-recalculate is disabled.
  - Surface computation errors near the relevant chart instead of only metadata chips.
  - Keep chart controls disabled while their backing data is unavailable or stale.
- [x] Improve array response chart presentation:
  - Add clear labels for absolute SPL vs normalized/directivity-only views.
  - Add receiver position, source count, active filter/preset, and air attenuation state in a compact summary.
  - Add optional normalized mode for comparing shapes without hiding absolute SPL mode.
- [ ] Improve polar and 3D balloon controls:
  - Make frequency selection consistent between polar and balloon views.
  - Add preset frequency buttons or a logarithmic slider for common bands.
  - Clarify when data is interpolated, mirrored by symmetry, or unavailable.
- [ ] Add visual QA checks:
  - Desktop and mobile screenshots for the visualization tab.
  - Verify charts and 3D canvas are nonblank after recomputation.
  - Check long labels and narrow layouts for overlap.

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
