# Response Data: Storage, Coordinates, Symmetry, and Display

This document explains how directivity response data is stored in GLL files, how the coordinate system is defined, how symmetry is encoded, and how the web demo uses these details when displaying responses and balloons.

This repository is based on reverse engineering and empirical validation. Where the format is ambiguous, this document calls out assumptions explicitly.

## 1) Where the response data lives in a GLL file

Directivity response data is stored inside the **BalloonData** structure for each source definition.

Key fields (from `docs/format.md`):

- `flags` (int32): bit 0 stored boolean (see section 3).
- `AngularResolution` (ResolutionDescriptor): defines the angular grid.
- `responseCount` (int32): number of stored transfer functions (responses).
- `responses[]`: sequence of transfer functions (TransferFunctionLsPs for version 1, CLogSpectrumLP for version 0).

### BalloonData layout (high level)

- Balloon block header (size + version check + sub version)
- `flags`
- `AngularResolution`
- `responseCount`
- `responses` (stored immediately after, at `ResponsesOffset`)

The response data is **not** interleaved with other data. It is a contiguous list of transfer functions, one per measurement angle, in a fixed order (see section 5).

## 2) Transfer function data format

Two formats exist, chosen by `BalloonData.ResponseVersion`:

- **Version 0: CLogSpectrumLP (legacy)**
  - Level and phase arrays stored directly (int16 values), optionally BitCompression encoded.

- **Version 1: TransferFunctionLsPs (current)**
  - Level and phase stored in a nested `ComplexSequence` with `Record` blocks.
  - Each `Record` may be raw int16 or BitCompression encoded.

After decompression:

- **Level values** are scaled by `0.01` → dB.
- **Phase values** are scaled by `0.001` → radians.

Frequency grid is defined by `LogSpectrumDefinition`:

```
frequency[i] = LowestFrequency * 2^(i / BandsPerOctave)
```

See `docs/format.md` for the precise binary layout and BitCompression algorithm.

## 3) BalloonData flags (detail)

`flags` is a 32-bit bitfield. Only **bit 0** is used and stored as a single boolean.

- **bit 0 (value 1)**: stored boolean (labeled "Interpolation" in our CLI/UI)

This flag is persisted but does not affect calculations; it is only serialized and round-tripped.

## 4) AngularResolution (detail)

`AngularResolution` is a nested block inside `BalloonData`. It has its own block wrapper and fixed size:

```text
int32  block_size        // must be 32
int16  version_check     // must be 0
int16  sub_version       // currently 0
int32  symmetry_code     // see mapping below
int32  front_half_flags  // bit 0 == front_half_only
double meridian_step     // degrees, horizontal step
double parallel_step     // degrees, vertical step
```

Notes:

- `front_half_only` is stored as a 32-bit flags field; only bit 0 is used.
- `meridian_step` and `parallel_step` are `float64` degrees.
- A `block_size` other than 32 or a `version_check` other than 0 is treated as invalid by the parser.

### Symmetry code mapping

The on-disk symmetry code is **not** the enum order. The mapping is:

```text
file symmetry_code -> enum
0 -> Axial
1 -> Quarter
2 -> Vertical
3 -> Horizontal
4 -> None
```

Enum order in code is: `None, Vertical, Horizontal, Quarter, Axial`.

## 5) Coordinate system and storage order

### Coordinate system

The format defines an angular grid using two axes:

- **Meridian** = horizontal axis = **Azimuth** (0-360 degrees)
- **Parallel** = vertical axis = **Elevation** (0-180 degrees)

This is consistent with:

- `ResolutionDescriptor.MeridianStep` described as "Horizontal angle step (degrees)"
- `ResolutionDescriptor.ParallelStep` described as "Vertical angle step (degrees)"
- Empirical checks on real GLL files (e.g., TiRAY-V1_3.gll in `testdata/gll/`)

### Indexing order

Responses are stored in **row-major order** where **parallel changes slowest**:

```go
index = parallelIdx * meridianCount + meridianIdx
```

This ordering is used in multiple places:

- `cmd/gllinfo/cmd/acoustic.go` (CSV export of angles)
- `web/app.js` (angle lookup in `getResponseWithSymmetry` and `computeResponseAngles`)

Given `MeridianStep` and `ParallelStep`, a response index maps to angles as:

```go
meridianIdx = index % meridianCount
parallelIdx = index / meridianCount

azimuthDeg  = meridianIdx * MeridianStep
parallelDeg = parallelIdx * ParallelStep
```

### Grid sizing derived from symmetry and front-half

The **measured** grid size is computed from symmetry and `front_half_only`:

- **Meridian points** (`GetMerPoints()`):
  - `None`: `360 / meridian_step`
  - `Vertical` or `Horizontal`: `180 / meridian_step + 1`
  - `Quarter`: `90 / meridian_step + 1`
  - `Axial`: `1`

- **Parallel points** (`GetParPoints()`):
  - `180 / parallel_step + 1`
  - If `front_half_only == true`: `90 / parallel_step + 1`

These counts define the **measured grid**, which may be much smaller than a
full 360×180 sphere.

### Response count can be smaller than the full grid

A full grid would be:

```go
fullMeridianCount = 360 / MeridianStep
fullParallelCount = 180 / ParallelStep + 1
fullTotal = fullMeridianCount * fullParallelCount
```

However, many files store **only a subset** of the theoretical grid. Example:

- TiRAY-V1_3.gll (source TiRAY LF)
  - `MeridianStep = 5`, `ParallelStep = 5`
  - Full grid would be 72 x 37 = 2664 points
  - Actual `responseCount = 667`
  - Implies: full azimuth coverage, but only 0-45 degrees elevation measured

Because of this, any code that assumes a full grid will produce wrong angles or missing data. Display code must infer the measured grid from `responseCount`.

### Pole de-duplication (important for response count)

The format uses a **collapsed pole representation**:

- At **parallel index 0** (front pole), all meridians map to a single point.
- At **parallel index max** (rear pole), all meridians map to a single point
  **only if** `front_half_only == false`.

This reduces the total point count from a full grid by:

- `merPoints - 1` when front-half-only
- `2 * (merPoints - 1)` when full sphere

This explains why `responseCount` is often smaller than
`meridianCount * parallelCount` even when the measured ranges appear complete.

## 6) Symmetry encoding

`ResolutionDescriptor.Symmetry` is an integer flag that indicates how the measured data can be mirrored.

Current naming (as used by `gllinfo` and the web UI). These are the enum
values; the **on-disk** numeric mapping is listed in section 4.

```plain
0 = None
1 = Vertical
2 = Horizontal
3 = Quarter
4 = Axial
```

`FrontHalfOnly` indicates that only the front hemisphere is measured.

### Actual symmetry behavior

Symmetry is applied by transforming the **meridian angle** (azimuth) before indexing into stored data:

- **Axial**: azimuth is forced to 0 (rotational symmetry).
- **Vertical**: if azimuth ≥ 180°, mirror to 360° − azimuth.
- **Quarter**: fold 360° into 0–90° by successive mirroring:
  - if azimuth ≥ 270° → 360° − azimuth
  - else if azimuth ≥ 180° → azimuth − 180°
  - else if azimuth ≥ 90° → 180° − azimuth
- **Horizontal**: meridian angles are **offset by 90°** when generating the
  angle list, and lookup mirrors around that rotated axis.

Parallel (elevation) handling is governed by `front_half_only`:

- If `front_half_only == true`, parallel angles above 90° are rejected.

### Interpretation (display-oriented)

The web demo mirrors only when symmetry or `front_half_only` allows it, and
leaves missing areas empty for `Symmetry == None`.

## 7) How the web demo uses this data

### 7.1 Response index → angle chips

`computeResponseAngles` in `web/app.js` uses the index ordering described above and the inferred grid dimensions to show the current response’s azimuth/elevation in the UI chips.

### 7.2 Polar plots

Polar plots display a single frequency slice from the response grid.

- **Horizontal plot**:
  - Scans azimuth 0-360 at elevation (parallel) = 0 degrees (on-axis)
  - This choice avoids missing data when elevation range is limited

- **Vertical plot**:
  - Scans elevation at azimuth (meridian) = 0 degrees (front axis)
  - If vertical mirroring is allowed by symmetry, the limited elevation range is mirrored to form a full 360-degree polar plot
  - If symmetry does not allow mirroring, the back half is left empty

### 7.3 3D balloon

The balloon mesh is built from a regular azimuth/elevation grid, but the **data values are taken from the response list** using symmetry-aware lookup.

Key behaviors:

- The grid size is inferred from `responseCount` and the resolution steps; it is not forced to the full 72x37 grid.
- When symmetry is declared, missing angles are mirrored based on the rules above.
- When no symmetry is declared, areas without measurements remain blank (rendered as null/gray).

This prevents the demo from visually “inventing” data when the file does not declare a symmetry.

## 8) Known ambiguities and constraints

- The symmetry names and on-disk numeric mapping are verified through reverse engineering.
  What remains ambiguous is the exact intended meaning of "Horizontal" vs "Vertical" beyond the transformations shown above.
- Some files report symmetry but still contain a partial grid (e.g., full azimuth + limited elevation). The display logic therefore must honor **both** symmetry and actual `responseCount`.
- If future research clarifies symmetry semantics, the mirroring rules should be updated accordingly.

## 9) Practical guidance

- When validating a file, check `responseCount` vs. full grid size.
- Always derive angle indices using `responseCount` and the resolution steps, not symmetry alone.
- Only mirror data when symmetry or `FrontHalfOnly` explicitly allows it.

---

## 10) Array Response Calculation (Line Arrays / Clusters)

This section describes how to calculate combined frequency responses for line array configurations.

### Overview

The calculation combines multiple acoustic sources with their:

- 3D positions and orientations
- Individual directivity balloons (frequency-dependent)
- Applied filters (internal crossovers, external EQ)
- Per-element gain settings

All sources are **coherently summed** in the complex domain, meaning phase interference between sources is accurately modeled.

### Calculation Flow

```text
┌─────────────────────────────────────────────────────────────────────┐
│                       ComputeSystemResponseAt                       │
│                 (Entry point for array calculation)                 │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │   For each box/element:   │
                    │ ComputeComponentResponseAt│
                    └─────────────┬─────────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │   BoxType.GetResponseAt   │
                    │  (Core calculation logic) │
                    └─────────────┬─────────────┘
                                  │
         ┌────────────────────────┼────────────────────────┐
         │                        │                        │
         ▼                        ▼                        ▼
┌──────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│Source.GetResponse│    │Element.GetSum   │    │Distance/Air     │
│At (Directivity)  │    │FilterSpectrum   │    │Attenuation      │
└────────┬─────────┘    └────────┬────────┘    └────────┬────────┘
         │                       │                      │
         └───────────────────────┼──────────────────────┘
                                 │
                     ┌───────────▼───────────┐
                     │  Complex Multiply:    │
                     │  Response × Filter    │
                     │  × OnAxis × (1/r)     │
                     └───────────┬───────────┘
                                 │
                     ┌───────────▼───────────┐
                     │  Complex Sum (Add)    │
                     │  All elements         │
                     │  with phase alignment │
                     └───────────────────────┘
```

### Key Components

#### ComputeSystemResponseAt

**Purpose:** Calculate the total array response at a given receiver position.

**Algorithm:** See `pkg/gll/array_calc.go`

```go
func ComputeSystemResponseAt(config *ArrayConfig, receiver Vector3D,
    airProps AirProperties, airAttenOn bool) *TransferFunction {

    var arraySpectrum *TransferFunction

    for i := range config.Elements {
        elem := &config.Elements[i]

        // Compute response for this element
        response, arrivalTime := computeElementResponseAt(elem, receiver, airProps, airAttenOn)
        if response == nil {
            continue
        }

        // Add arrival time delay (time-align to acoustic center)
        response.AddDelay(arrivalTime)

        if arraySpectrum == nil {
            arraySpectrum = response
        } else {
            arraySpectrum.Add(response) // Complex (coherent) sum
        }
    }
    return arraySpectrum
}
```

**Key insight:** The `Add()` method performs **coherent complex summation** - it converts to Real/Imaginary representation and sums, preserving phase relationships for accurate interference modeling.

#### computeElementResponseAt

**Purpose:** Calculate response for a single cabinet at a receiver position.

```go
func computeElementResponseAt(elem *ArrayElement, receiver Vector3D,
    airProps AirProperties, airAttenOn bool) (*TransferFunction, float64) {

    var boxSpectrum *TransferFunction

    for sourceIdx, srcDef := range elem.SourceDefs {
        // Get directivity response at receiver angle
        response, onAxis, propagationFactor := getSourceResponseAt(
            srcDef, elem.Position, elem.Angles, receiver, airProps)

        // Apply filter (complex multiply)
        if sourceIdx < len(elem.FilterSpectra) && elem.FilterSpectra[sourceIdx] != nil {
            response.Multiply(elem.FilterSpectra[sourceIdx])
        }

        // Apply element gain
        response.AddGain(elem.Gain)

        // Apply on-axis normalization
        if onAxis != nil {
            response.Multiply(onAxis)
        }

        // Apply distance attenuation (1/r spherical spreading)
        response.AddGain(20.0 * math.Log10(propagationFactor))

        // Coherent sum for multi-way speakers
        if boxSpectrum == nil {
            boxSpectrum = response
        } else {
            boxSpectrum.Add(response)
        }
    }

    // Calculate arrival time
    distance := vectorLength(vectorSub(receiver, elem.Position))
    arrivalTime := distance / airProps.Speed

    return boxSpectrum, arrivalTime
}
```

**Filter types:**

| Filter         | Source                       | Description                       |
| -------------- | ---------------------------- | --------------------------------- |
| InternalFilter | FilterGroup/FilterDefinition | Passive crossover, factory EQ     |
| ExternalFilter | User configuration           | External DSP, EQ curves           |
| Gain           | Element setting              | Per-element level adjustment (dB) |

#### getSourceResponseAt

**Purpose:** Get directivity response for a single source at a given angle.

```go
func getSourceResponseAt(srcDef *SourceDefinition, sourcePos, sourceAngles, receiver Vector3D,
    airProps AirProperties) (*TransferFunction, *TransferFunction, float64) {

    // Calculate vector from source to receiver
    vec := vectorSub(receiver, sourcePos)

    // Calculate distance and propagation factor (1/r)
    distance := vectorLength(vec)
    if distance < 0.01 {
        distance = 0.01
    }
    propagationFactor := 1.0 / distance

    // Convert to spherical angles relative to source orientation
    theta, phi := getThetaPhi(vec, sourceAngles)

    // Get response at that angle (with interpolation)
    response := srcDef.BalloonData.GetResponseAtAngle(theta, phi)

    // Get on-axis response for normalization
    onAxis := srcDef.BalloonData.GetResponseAtAngle(0, 0)

    // Add propagation delay
    response.AddDelay(distance / airProps.Speed)

    return response, onAxis, propagationFactor
}
```

### Complex Spectrum Operations

#### Addition (Coherent Sum)

The `Add()` method performs coherent summation by converting to Real/Imaginary representation:

```go
func (tf *TransferFunction) Add(other *TransferFunction) {
    real1, imag1 := tf.ToComplex()
    real2, imag2 := other.ToComplex()

    for i := range real1 {
        real1[i] += real2[i]
        imag1[i] += imag2[i]
    }

    tf.FromComplex(real1, imag1)
}
```

This preserves phase information, allowing accurate modeling of constructive/destructive interference between array elements.

#### Multiplication (Filter Application)

The `Multiply()` method applies filters in the Level/Phase domain:

```go
func (tf *TransferFunction) Multiply(filter *TransferFunction) {
    for i := range tf.Level {
        tf.Level[i] += filter.Level[i]  // Add levels (dB)
        tf.Phase[i] += filter.Phase[i]  // Add phases (radians)
    }
}
```

#### Delay Addition

The `AddDelay()` method modifies phase based on frequency:

```go
func (tf *TransferFunction) AddDelay(delay float64) {
    tf.Delay += delay
    for i := range tf.Phase {
        freq := tf.Definition.GetFrequency(i)
        tf.Phase[i] += 2.0 * math.Pi * freq * delay
    }
}
```

### Air Absorption

When enabled, air absorption loss is applied per frequency band:

```go
if airAttenOn {
    distance := 1.0 / propagationFactor
    for band := 0; band < len(response.Level); band++ {
        freq := response.Definition.GetFrequency(band)
        response.Level[band] -= distance * airProps.GetAirLossPerMeter(freq)
    }
}
```

The air loss follows ISO 9613 standards and depends on temperature, humidity, and frequency.

### Source Grouping Optimization

Sources within 0.5m of each other can be grouped into "Centers" for faster far-field calculations:

```go
// Check if sources are close enough to group
maxDistance := 0.0
for k := 0; k < len(sources); k++ {
    for l := k + 1; l < len(sources); l++ {
        dist := vectorLength(vectorSub(sources[k].Position, sources[l].Position))
        if dist > maxDistance {
            maxDistance = dist
        }
    }
}
shouldGroup := maxDistance < 0.5 // 0.5 meter threshold
```

### Data Flow Summary

```text
GLL File
    │
    ├── SourceDefinition
    │       └── BalloonData (directivity measurements)
    │               └── []TransferFunction (per angle × frequency)
    │
    ├── FilterGroup
    │       └── FilterDefinition[] (crossover/EQ curves)
    │               └── GenericFilterBank
    │
    └── ClusterSetup / ArrayConfig
            └── ArrayElement[] (position, angle, filter selection, gain)
                    │
                    ▼
            ┌───────────────────┐
            │ Runtime Calculation│
            │                   │
            │ For each receiver:│
            │ 1. Get angles     │
            │ 2. Interpolate    │
            │ 3. Apply filters  │
            │ 4. Apply distance │
            │ 5. Sum coherently │
            └───────────────────┘
                    │
                    ▼
            Combined Response
            (Level + Phase per frequency)
```

### Implementation Notes

1. **Caching:** `Element.GetSumFilterSpectrum` caches computed filter spectra to avoid recalculation.

2. **Frequency Resolution:** Calculations use `LogSpectrumDefinition` with configurable bands (typically 1/12 octave or finer).

3. **Coordinate Transform:** Receiver positions are transformed into each source's local coordinate system before angle calculation.

4. **Precision:** For multi-source summation, spectra use `TransferFunctionRdId` (Real/Imaginary with Delay) for maximum phase accuracy.

5. **On-Axis Response:** The on-axis response (`cOnAxis`) is multiplied separately to normalize the directivity pattern.

---

If you want this document to also describe the exact binary offsets or reader code paths, we can extend it with references into `pkg/gll/source_parse.go` and `pkg/gll/transfer_parse.go`.
