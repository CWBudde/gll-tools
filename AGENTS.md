# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gll-tools is a Go library and CLI toolset for parsing EASE GLL (Generic Loudspeaker Library) files on Linux. GLL is a proprietary binary format used by AFMG's EASE acoustic simulation software to describe loudspeaker systems.

**Key Facts:**

- Language: Go 1.25
- Organization: cwbudde on GitHub
- Primary Purpose: Reverse-engineer and extract data from proprietary GLL binary files
- No Windows/EASE software required

## Build, Test, and Development Commands

The project uses `just` (justfile) for task automation. Common commands:

```bash
# Development workflow
just fmt                    # Format code (uses treefmt)
just lint                   # Run golangci-lint
just lint-fix              # Run golangci-lint with auto-fix
just test                  # Run all tests
just test-coverage         # Generate coverage report
just check                 # Run all checks (fmt + lint + test + tidy)

# Building
just build                 # Build all CLI tools (outputs to bin/)
just build-gllinfo         # Build just gllinfo
just install               # Install tools to $GOPATH/bin

# Running tools directly
just test-sample           # Run gllinfo on default sample file
just extract-sample        # Extract resources from sample file
just parse-xgll            # Parse XGLL text format file
just convert-xgll          # Convert XGLL to binary container

# Standard Go commands also work
go build ./cmd/gllinfo     # Build gllinfo
go build ./cmd/xgllc       # Build xgllc (XGLL compiler)
go test ./...              # Run all tests
go test ./pkg/gll          # Test specific package
```

## Architecture

### Package Structure

```
gll-tools/
├── cmd/
│   ├── gllinfo/          # CLI for inspecting/extracting GLL files
│   │   └── cmd/          # Cobra subcommands (info, extract, acoustic, version)
│   ├── xgllc/            # CLI for parsing/converting XGLL text format
│   │   └── cmd/          # Cobra subcommands (parse, validate, convert)
│   └── gllwasm/          # WebAssembly build for browser-based parsing
├── pkg/
│   ├── gll/              # Public library for GLL binary parsing
│   └── xgll/             # Public library for XGLL text format parsing
├── internal/gll/         # Internal parsing utilities (ByteReader, BitCompression)
├── docs/                 # Format specifications and research
├── testdata/             # Test GLL and XGLL files
└── legacy/viewer/        # Extracted EASE GLL Viewer (for reverse engineering)
```

### Core Architecture Concepts

**Binary Format Parsing (`pkg/gll`):**

- All GLL files start with `EGLL` magic bytes
- Uses little-endian byte order throughout
- Strings are length-prefixed (uint16 length + UTF-8 data)
- Blocks are length-prefixed (int32 size + version check + content)
- Version-aware parsing: format versions 3-6 supported, structures have sub-versions

**Key Data Flow:**

1. `Parse()` reads header → GenSystem → scans for embedded resources
2. Database parsing is lazy - only reads buffers when accessed
3. ByteReader (`internal/gll/reader.go`) wraps io.ReadSeeker and tracks offsets
4. BitCompression (`internal/gll/bitcompression.go`) decompresses acoustic response data

**Major Structures:**

- **GenSystem**: Root container with metadata (manufacturer, model, description)
- **Database**: Container for all component data (BoxTypes, SourceDefinitions, FilterGroups, etc.)
- **SourceDefinition**: Acoustic source with directivity balloon data and frequency responses
- **BalloonData**: Directivity measurements with 72×37 grid (5° resolution)
- **TransferFunction**: Frequency/phase response data with BitCompression encoding

**XGLL Text Format (`pkg/xgll`):**

- Human-readable text representation of GLL binary format
- Lexer/parser converts to AST
- Can convert to binary containers (xgllbin format or minimal GLL)
- Supports validation against system-specific block schemas

### Important Implementation Details

**Version Handling:**

- Header format versions: 3, 4 (adds checksum), 6 (adds SHA256 hash)
- Database has sub_version field for feature versioning
- Individual structures (BoxType, FilterGroup, etc.) have their own sub_versions
- Always check version before reading version-specific fields

**Resource Extraction:**

- Embedded PNGs detected by signature (`\x89PNG\r\n\x1a\n`)
- Zlib-compressed blocks start with `0x78` byte
- DataFiles buffer contains embedded geometry (.xed files) and images
- Resources can be extracted with automatic decompression

**Acoustic Data:**

- Two transfer function formats: CLogSpectrumLP (v0, legacy) and TransferFunctionLsPs (v1, current)
- BitCompression algorithm uses variable bit-depth with delta encoding
- Response data scaled: levels in 0.01 dB units, phase in 0.001 radian units
- Balloon grid: meridian (0-360°, 72 points) × parallel (0-180°, 37 points)

**Testing:**

- Golden file tests in `pkg/gll/golden_test.go` compare against expected JSON output
- Test data in `testdata/gll/` (3 sample files) and `legacy/viewer/` (10 files)
- Use table-driven tests for parsing variations

## Format Documentation

**Critical Reference:** `docs/format.md` contains the complete reverse-engineered binary format specification. Always consult this when implementing new structure parsing.

Key sections:

- File header structure (magic, version, checksum, hash)
- Block structure pattern (size + version_check + sub_version + content)
- Database component order and dependencies
- Individual structure specifications (BoxType, SourceDefinition, FilterGroup, Limit, Warning, etc.)
- Transfer function formats and BitCompression algorithm

## Common Development Patterns

**Adding New Structure Parsing:**

1. Document structure in `docs/format.md` first
2. Add Go type definition to `pkg/gll/types.go`
3. Implement parsing function in appropriate file (e.g., `database.go`, `source.go`)
4. Use ByteReader methods: `ReadInt16()`, `ReadInt32()`, `ReadDouble()`, `ReadString()`
5. Always check `version_check == 0` before parsing (unless documented otherwise)
6. Check sub_version before reading version-specific fields
7. Add test case with sample GLL file

**BitCompression Decompression:**

- Used for frequency/phase response arrays
- See `internal/gll/bitcompression.go` for algorithm
- Input: compressed byte stream with bit depth parameter
- Output: int32 array with delta decoding applied

**CLI Subcommand Pattern (Cobra):**

- Root command in `cmd/<tool>/cmd/root.go`
- Subcommands in separate files: `cmd/<tool>/cmd/<subcommand>.go`
- Use `--json` flag for machine-readable output
- Use `-v/--verbose` for detailed human-readable output

## Working with Legacy Code

**Usage:**

- Check legacy code when implementing new structures
- Look for `SaveToTarget` methods to understand serialization
- Compare field order and types
- Note version checks and conditional serialization
- Don't copy code directly - write clean Go implementations

## Project Status (Phase 4 Nearly Complete)

**Implemented:**

- ✅ Header parsing (all versions 3-6)
- ✅ GenSystem metadata extraction
- ✅ Resource scanning and extraction (PNG, zlib, fonts, PDFs)
- ✅ Database parsing: BoxTypes, SourceDefinitions, FilterGroups, Limits, Warnings
- ✅ Transfer function parsing with BitCompression decompression
- ✅ XGLL text format parsing and conversion

**In Progress/Future:**

- ⏳ Remaining database buffers: Connectors, Frames, ClusterSetups, Presets
- ⏳ CSV export for response data
- ⏳ CLF/FRD/SOFA format export for acoustic data
- ⏳ WebAssembly demo deployment

See [PLAN.md](PLAN.md) for complete development roadmap.

## Legal Context

This project is for educational and interoperability purposes. GLL is a proprietary format owned by AFMG. The project does not redistribute any AFMG software or copyrighted material - only reverse-engineered format specifications and clean-room implementations.

## Releasing, and Not Drifting

This module is part of the `github.com/cwbudde/algo-*` family, which is co-developed
across separate repositories. That arrangement failed once already, and the rules below
exist to stop it failing the same way twice.

**What went wrong (August 2026).** The family had drifted onto three different `algo-fft`
versions simultaneously — `algo-pde` on v0.6.15, `algo-dsp` on v0.7.3, `algo-acoustics` on
v0.6.11 — while `algo-fft`'s own `main` sat 97 commits past its latest tag and its
CHANGELOG documented a `v0.7.5` that had never been tagged. Because `algo-fft`'s generic
`PlanReal2D`/`PlanReal3D` had changed signature between the v0.6 and v0.7 lines, _no single
upgrade anywhere would compile_. Untangling it took a day and four coordinated releases.

Three separate mistakes combined to produce that. Each now has a check.

### 1. Do not let work pile up untagged

Work that only exists on `main` cannot be consumed. If you finish something a sibling repo
needs, tag it — do not wait for a milestone.

```bash
just check-unreleased     # how much is sitting past the latest tag?
```

A scheduled CI job (`.github/workflows/dep-drift.yml`) reports this weekly.

### 2. Do not sit on stale siblings

```bash
just check-deps           # are all github.com/cwbudde/* deps at their latest tags?
```

This is wired into the repo's aggregate check recipe, and the same scheduled job files a
GitHub issue when it starts failing. If a bump is _deliberately_ deferred, write down why in
`PLAN.md` — an undocumented old pin is indistinguishable from a forgotten one.

Renovate (`.github/renovate.json`) opens the bump PRs automatically and groups the whole
`cwbudde` family into a single PR on purpose: an incompatible `algo-fft` can reach a
consumer through two different dependency paths at once, so bumping them one PR at a time
produces intermediate combinations that never build.

### 3. Never remove or change exported API without the version saying so

Always release through the guard rather than by hand:

```bash
just tag-release v0.8.0       # runs every precondition, then tags and pushes
```

It refuses to tag when the tree is dirty, when `HEAD` is not a pushed default branch, when the tag
already exists or does not sort after the current one, when siblings are stale, when
`CHANGELOG.md` has no section for the version, or when the exported API changed
incompatibly without the version reflecting it.

**That last rule is stricter than semver, deliberately.** Semver exempts `v0.x` — "anything
MAY change at any time" — so `gorelease` will happily approve a _patch_ bump across a
removed symbol. Every module in this family is `v0.x`, so that exemption is exactly the hole
we fell through: `KernelEightStep` was removed and `PlanReal2D` became generic, and nothing
in the version numbers said so. The guard therefore requires a **minor** bump for any
incompatible change while on `v0.x`.

When you do break API, say so in the CHANGELOG in the form a consumer needs: the old
signature, the new signature, and the call-site rewrite. "Refactored plans" does not help
anyone; `NewPlanReal2D(rows, cols)` → `NewPlanReal2D64(rows, cols)` does.

### Order of operations for a cross-repo change

Releases must flow up the dependency graph, never sideways:

```
algo-vecmath ─┐
algo-approx  ─┼─→ algo-dsp ─┐
algo-fft ─────┴─→ algo-pde ─┴─→ algo-acoustics
```

Tag the dependency first, then bump and tag its consumers, then the consumers' consumers.
Bumping a consumer before its dependency is tagged is what forces pseudo-versions into
`go.mod`, and those are how a repo quietly ends up pinned to a commit nobody can find later.
