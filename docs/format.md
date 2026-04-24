# GLL File Format Specification (Reverse Engineered)

This document describes the GLL (Generic Loudspeaker Library) file format.

## Overview

GLL files are a proprietary binary format developed by AFMG for their EASE acoustic simulation software. Despite the `.gll` extension suggesting a DLL, these are **NOT** Windows DLLs - they are a custom container format called "EGLL".

The format uses:

- Little-endian byte order
- Length-prefixed strings (2-byte uint16 length + UTF-8 data)
- Length-prefixed blocks (4-byte int32 size at block start)
- Versioned data structures for forward/backward compatibility

---

## File Header

### Primary Header Structure

| Offset | Size | Type    | Value/Description                                |
| ------ | ---- | ------- | ------------------------------------------------ |
| 0x00   | 4    | char[4] | `EGLL` - File magic identifier                   |
| 0x04   | 4    | int32   | Reserved (always 0)                              |
| 0x08   | 2+N  | string  | `EASE_GLL` - Format identifier (length-prefixed) |
| varies | 2    | int16   | Format version (3-6, current is 6)               |
| varies | 2    | int16   | Sub-version                                      |

### Checksum Block (version >= 4)

| Field       | Size | Description                            |
| ----------- | ---- | -------------------------------------- |
| checksum[0] | 1    | Checksum byte 0                        |
| checksum[1] | 1    | Checksum byte 1                        |
| checksum[2] | 1    | Checksum byte 2                        |
| checksum[3] | 1    | Checksum byte 3 (XOR of 0-2 with 0x37) |

### Hash ID Block (version >= 6)

| Field      | Size | Description                      |
| ---------- | ---- | -------------------------------- |
| hashLength | 4    | Length of hash (always 32)       |
| hashBytes  | 32   | SHA256 hash of GenSystem content |

---

## String Encoding

All strings are length-prefixed with UTF-8 encoding:

```text
[uint16 LE: byte length]
[N bytes: UTF-8 string data]
```

Note: The length is in bytes, not characters.

---

## Block Structure

All major data blocks use a common wrapper:

```text
[int32: block_size]           // Total size including this field
[int16: version_check]        // Must be 0 for current readers
[int16: sub_version]          // Feature version within block
[...block content...]
```

The reader can skip unknown blocks by seeking `block_size` bytes forward.

---

## GenSystem (Main Container)

The GenSystem is the root data structure containing all loudspeaker data.

### GenSystem Header

| Field           | Type   | Description                            |
| --------------- | ------ | -------------------------------------- |
| block_size      | int32  | Total block size                       |
| version_check   | int16  | Must be 0                              |
| sub_version     | int16  | Feature version (0-1)                  |
| Label           | string | Display name                           |
| Version         | double | GLL version number                     |
| Key             | string | Internal identifier                    |
| Type            | int32  | 0=LineArray, 1=Cluster, 2=Loudspeaker  |
| Company         | string | Manufacturer name                      |
| InfoText        | string | Product description                    |
| CopyrightText   | string | Copyright notice                       |
| SupportText     | string | Support information                    |
| WebsiteText     | string | Manufacturer website URL               |
| EmailText       | string | Contact email                          |
| BackgroundColor | int32  | RGB color for UI                       |
| Database        | block  | Database containing all component data |

### GenSystem Flags (sub_version >= 1)

| Field | Type  | Description                                                    |
| ----- | ----- | -------------------------------------------------------------- |
| flags | int32 | Bit 0: AllowUserDefinedClusterSetup, Bit 1: EnableForSubArrays |

---

## Database Structure

The Database contains all component buffers.

### Database Loading Order

| Order | Component         | Description                           |
| ----- | ----------------- | ------------------------------------- |
| 1     | DataFiles         | Embedded binary data files            |
| 2     | BoxTypes          | Speaker cabinet types                 |
| 3     | Frames            | Rigging frames                        |
| 4     | Connectors        | Connection definitions                |
| 5     | Limits            | Mechanical/electrical limits          |
| 6     | SourceDefinitions | Acoustic source data                  |
| 7     | Warnings          | Configuration warnings                |
| 8     | FilterGroups      | DSP filter sets                       |
| 9     | ClusterSetups     | Predefined array configurations       |
| 10    | Presets           | System presets (version >= 1)         |
| 11    | IncludeFiles      | Additional data files (version >= 2)  |
| 12    | AuthorFiles       | Authorization data (version >= 2)     |
| 13    | Transformers      | Power transformer data (version >= 3) |

---

## BoxType (Speaker Cabinet)

Represents a physical speaker cabinet type.

| Field                  | Type                 | Description                                  |
| ---------------------- | -------------------- | -------------------------------------------- |
| Label                  | string               | Display name                                 |
| Key                    | string               | Internal identifier                          |
| Sources                | SourceBuffer         | Acoustic sources in this box                 |
| InputConfigs           | BoxInputConfigBuffer | Input configurations                         |
| CaseGeometry           | Geometry             | 3D model (see CaseGeometry section below)    |
| NextPivot              | Vector3D             | Pivot point for next box in array            |
| ReferencePoint         | Vector3D             | Acoustic reference point                     |
| CenterOfMass           | Vector3D             | Center of mass                               |
| Weight                 | double               | Weight in kg                                 |
| VerticalOpeningAngle   | double               | Vertical coverage angle (sub_version >= 1)   |
| HorizontalOpeningAngle | double               | Horizontal coverage angle (sub_version >= 1) |

---

### SourceBuffer (BoxType Sources)

Defines the acoustic source placements inside a cabinet.

This buffer follows the standard ObjectLSCBuffer wrapper:

| Field      | Type   | Description       |
| ---------- | ------ | ----------------- |
| block_size | int32  | Total buffer size |
| vcheck     | int16  | Must be 0         |
| sver       | int16  | Sub-version       |
| count      | int32  | Number of sources |
| items[]    | Source | Source entries    |

#### Source

| Field        | Type     | Description                           |
| ------------ | -------- | ------------------------------------- |
| block_size   | int32    | Total block size                      |
| vcheck       | int16    | Must be 0                             |
| sver         | int16    | Sub-version                           |
| SourceDefKey | string   | Key of SourceDefinition to use        |
| Position     | Vector3D | X, Y, Z position (3 doubles)          |
| Angles       | Vector3D | H, V, R rotation (3 doubles, radians) |
| Label        | string   | Display name                          |
| Key          | string   | Source identifier                     |

---

## Frame (Rigging Frame)

Represents a rigging frame for line array configurations.

**Note:** Frame uses `vcheck=1` (not 0 like most structures).

| Field        | Type                  | Description                               |
| ------------ | --------------------- | ----------------------------------------- |
| block_size   | int32                 | Total block size                          |
| vcheck       | int16                 | Must be 1                                 |
| sver         | int16                 | Feature version (0 or 1)                  |
| Label        | string                | Display name                              |
| TypeFlown    | int32                 | 1=flown, 0=ground stacked                 |
| CaseGeometry | Geometry              | 3D model (see CaseGeometry section below) |
| [padding]    | int32 × 2             | Only if sver=0                            |
| NextPivot    | Vector3D              | Pivot point for next element (3 doubles)  |
| [padding]    | int32 × 2             | Only if sver=0                            |
| CenterOfMass | Vector3D              | Center of mass position (3 doubles)       |
| Weight       | double                | Weight in kg                              |
| PinPoints    | LabeledVector3DBuffer | Rigging attachment points                 |
| Key          | string                | Internal identifier (at end, not start!)  |

### Notes on Frame Parsing

- The Key field is at the **end** of the structure, unlike most other types
- The sver=0 format has padding fields before Vector3D reads (legacy compatibility)

---

## Connector (Box-to-Box Connection)

Defines connection points between speaker cabinets with available splay angles.

**Note:** Connector uses `vcheck=1` (not 0 like most structures).

| Field      | Type                | Description                         |
| ---------- | ------------------- | ----------------------------------- |
| block_size | int32               | Total block size                    |
| vcheck     | int16               | Must be 1                           |
| sver       | int16               | Feature version                     |
| UpperBox   | string              | Key of upper box type in the array  |
| LowerBox   | string              | Key of lower box type in the array  |
| Frame      | string              | Key of rigging frame (may be empty) |
| Angles     | LabeledValueDBuffer | Available splay angles              |

### LabeledValueD (Named Double Value)

Used for connector angles and other labeled numeric values.

| Field      | Type   | Description                   |
| ---------- | ------ | ----------------------------- |
| block_size | int32  | Total block size              |
| vcheck     | int16  | Must be 0                     |
| sver       | int16  | Feature version               |
| Label      | string | Human-readable label          |
| Value      | double | Numeric value (e.g., radians) |

### LabeledVector3D (Named 3D Point)

Used for pin points and other labeled 3D coordinates.

| Field      | Type      | Description                |
| ---------- | --------- | -------------------------- |
| block_size | int32     | Total block size           |
| vcheck     | int16     | Must be 0                  |
| sver       | int16     | Feature version (0 or 1)   |
| Label      | string    | Human-readable label       |
| [padding]  | int32 × 2 | Only if sver=0             |
| Vector     | Vector3D  | 3D coordinates (3 doubles) |

---

## BoxInputConfig (Input Channel Configuration)

Defines input channel wiring configurations for a box type.

| Field      | Type           | Description               |
| ---------- | -------------- | ------------------------- |
| block_size | int32          | Total block size          |
| vcheck     | int16          | Must be 0                 |
| sver       | int16          | Feature version           |
| Label      | string         | Display name              |
| Key        | string         | Internal identifier       |
| Inputs     | BoxInputBuffer | Input channel definitions |

### BoxInput (Input Channel)

| Field          | Type               | Description                      |
| -------------- | ------------------ | -------------------------------- |
| block_size     | int32              | Total block size                 |
| vcheck         | int16              | Must be 0                        |
| sver           | int16              | Feature version (0 or 1)         |
| Label          | string             | Input channel label              |
| LinkCount      | int32              | Number of source-filter links    |
| Links          | SourceFilterLink[] | Source to filter group mappings  |
| RatedImpedance | double             | Rated impedance in ohms (sver≥1) |

### SourceFilterLink (inline, no block wrapper)

| Field        | Type   | Description            |
| ------------ | ------ | ---------------------- |
| SourceKey    | string | Key of acoustic source |
| FilterGrpKey | string | Key of filter group    |

---

## SourceDefinitionItem

Wrapper for acoustic source data.

| Field      | Type             | Description        |
| ---------- | ---------------- | ------------------ |
| Key        | string           | Source identifier  |
| SourceDefX | SourceDefinition | Full acoustic data |

---

## SourceDefinition (Acoustic Data)

Contains complete acoustic measurement data for a driver/source.

### Header Fields

| Field                | Type   | Description                        |
| -------------------- | ------ | ---------------------------------- |
| Label                | string | Source name                        |
| CompanyLabel         | string | Manufacturer                       |
| Description          | string | Source description                 |
| NominalBandwidthFrom | double | Low frequency limit (Hz)           |
| NominalBandwidthTo   | double | High frequency limit (Hz)          |
| DataType             | int32  | 0=HighRes, 1=ThirdOctave, 2=Octave |

### Balloon Data

| Field                 | Type        | Description         |
| --------------------- | ----------- | ------------------- |
| BalloonCorrectionData | BalloonData | Directivity balloon |

### On-Axis Spectrum

| Field              | Type                 | Description                       |
| ------------------ | -------------------- | --------------------------------- |
| flags              | int32                | Bit 0: has OnAxisSpectrumFile     |
| reference_level    | double               | Reference level (typically 94 dB) |
| OnAxisSpectrumFile | CLinResponse?        | Optional file reference           |
| OnAxisSpectrumRaw  | TransferFunctionLsPs | On-axis frequency response        |

### Impedance Data

| Field          | Type                 | Description              |
| -------------- | -------------------- | ------------------------ |
| flags          | int32                | Bit 0: has ImpedanceFile |
| RatedImpedance | double               | Nominal impedance (ohms) |
| ImpedanceFile  | CLinResponse?        | Optional file reference  |
| Impedance      | TransferFunctionMfPi | Impedance curve          |

### Power Handling

| Field             | Type                 | Description            |
| ----------------- | -------------------- | ---------------------- |
| BroadbandMaxVolts | ISOBroadbandMaxVolts | Maximum voltage limits |

### Measurement Conditions (sub_version >= 2)

| Field                  | Type   | Description                             |
| ---------------------- | ------ | --------------------------------------- |
| MeasuredOnAxisVoltage  | double | Test voltage (default 2.828V = 1W@8ohm) |
| MeasuredOnAxisDistance | double | Measurement distance (default 1m)       |
| MeasuredOnAxisGainIndB | double | Additional gain correction              |

### Environmental Conditions (sub_version >= 3)

| Field       | Type   | Description                                          |
| ----------- | ------ | ---------------------------------------------------- |
| flags       | int32  | Bit 0: CompensateOnAxisLevel, Bit 1: CompensateDelay |
| Temperature | double | Temperature (deg Celsius, default 25)                |
| Humidity    | double | Relative humidity (%, default 60)                    |
| Pressure    | double | Atmospheric pressure (kPa, default 101.325)          |

### Measurement Info (sub_version >= 5)

| Field         | Type   | Description             |
| ------------- | ------ | ----------------------- |
| ContactPerson | string | Measurement contact     |
| EmailAddress  | string | Contact email           |
| Website       | string | Website                 |
| Environment   | string | Measurement environment |
| Notes         | string | Notes                   |
| DataOrigin    | string | Data origin             |
| DateTime      | int64  | .NET DateTime binary    |

### Additional Data (sub_version >= 6)

| Field                | Type   | Description            |
| -------------------- | ------ | ---------------------- |
| CompanyWebsite       | string | Company website        |
| RatedHorizontalAngle | double | Rated H coverage angle |
| RatedVerticalAngle   | double | Rated V coverage angle |

### Far-Field Directivity (version >= 5)

| Field               | Type   | Description                                            |
| ------------------- | ------ | ------------------------------------------------------ |
| directivityType     | int32  | 0=Point, 1=Line, 2=CircularPiston, 3=RectangularPiston |
| FarFieldDirectivity | varies | Type-specific parameters                               |

---

## BalloonData (Directivity)

Contains the directivity balloon - SPL response at all angles.

| Field             | Type                   | Description                  |
| ----------------- | ---------------------- | ---------------------------- |
| flags             | int32                  | Bit 0: interpolation enabled |
| AngularResolution | ResolutionDescriptor   | Angular grid definition      |
| responseCount     | int32                  | Number of transfer functions |
| responses[]       | TransferFunctionLsPs[] | Response at each angle       |

Implementation note: the binary structure is documented here, but the exact acoustic combination contract between `responses[]`, `OnAxisSpectrumRaw`, phase delay, and air attenuation is tracked separately in [docs/acoustic-model.md](acoustic-model.md).

### Angular Resolution

Standard resolution is 5-degree steps:

- 72 meridian points (0-355 degrees horizontal)
- 37 parallel points (0-180 degrees vertical)
- Total: 2664 measurement points

### Balloon.ResolutionDescriptor

| Field        | Type   | Description                                  |
| ------------ | ------ | -------------------------------------------- |
| block_size   | int32  | Total block size                             |
| vcheck       | int16  | Must be 0                                    |
| sver         | int16  | Sub-version (must be 0)                      |
| Symmetry     | int32  | Balloon symmetry type (see below)            |
| Flags        | int32  | Bit 0: FrontHalfOnly (only front hemisphere) |
| MeridianStep | double | Horizontal angle step (degrees), e.g. 5.0    |
| ParallelStep | double | Vertical angle step (degrees), e.g. 5.0      |

**Symmetry Types**:

| Value | Name       | MeridianPoints | Description                                            |
| ----- | ---------- | -------------- | ------------------------------------------------------ |
| 0     | Axial      | 1              | Rotationally symmetric (horn/subwoofer)                |
| 1     | Quarter    | 90°/step + 1   | Quarter-space symmetry (0-90° meridian)                |
| 2     | Vertical   | 180°/step + 1  | Left/right symmetric (0-180° meridian)                 |
| 3     | Horizontal | 180°/step + 1  | Top/bottom symmetric (90-270° meridian, offset by 90°) |
| 4     | None       | 360°/step      | Full sphere, no symmetry                               |

**Symmetry Mirroring Logic** (from `GetInterpolatedPoint`):

- **Axial**: All meridian values map to 0
- **Quarter**: Values 0-90° used directly; 90-180° → `180 - mer`; 180-270° → `mer - 180`; 270-360° → `360 - mer`
- **Vertical**: Values 0-180° used directly; 180-360° → `360 - mer`
- **Horizontal**: Meridian offset by -90°, then mirrored: negative → `|mer|`; 180-360° → `360 - mer`
- **None**: No mirroring applied

---

## TransferFunctionLsPs (Level/Phase Spectrum)

Complex frequency response stored as Level (dB) and Phase (radians).

| Field      | Type                  | Description            |
| ---------- | --------------------- | ---------------------- |
| Definition | LogSpectrumDefinition | Frequency grid         |
| Level[]    | float64[]             | Level in dB            |
| Phase[]    | float64[]             | Phase in radians       |
| Delay      | float64               | Group delay in seconds |

See "Transfer Function Storage Formats" below for binary encoding details.

---

## LogSpectrumDefinition

Defines the frequency grid for spectral data.

### Binary Format

| Field           | Type   | Description                      |
| --------------- | ------ | -------------------------------- |
| BandsPerOctave  | int32  | Frequency resolution (see below) |
| LowestFrequency | double | Lowest center frequency (Hz)     |
| NumberOfBands   | int32  | Total number of frequency bands  |

### Standard Resolutions

| Name     | BandsPerOctave | Bands | Start Freq | End Freq  |
| -------- | -------------- | ----- | ---------- | --------- |
| EQTones  | 24             | 241   | 22.1 Hz    | 22,627 Hz |
| EThirds  | 3              | 30    | varies     | varies    |
| EOctaves | 1              | 10    | varies     | varies    |

The center frequency for band index `i` is calculated as:

```text
freq[i] = LowestFrequency * 2^(i / BandsPerOctave)
```

---

## Transfer Function Storage Formats

GLL files use two different formats for storing transfer function data (level/phase responses), selected by the BalloonData version field.

### Version 0: CLogSpectrumLP (Legacy Format)

Used in older GLL files. Stores level and phase directly with optional compression.

| Field           | Type                  | Description               |
| --------------- | --------------------- | ------------------------- |
| block_size      | int32                 | Total block size          |
| version_check   | int16                 | 0 or 1                    |
| sub_version     | int16                 | Feature version           |
| Definition      | LogSpectrumDefinition | Frequency grid (16 bytes) |
| compressionType | int32                 | 0=raw, 1=BitCompression   |
| levelData       | (see below)           | Level values              |
| phaseData       | (see below)           | Phase values              |
| Delay           | double                | Group delay in seconds    |

**Uncompressed (compressionType=0):**

```text
[int32: levelCount]
[int16 * levelCount: level values]
[int32: phaseCount]
[int16 * phaseCount: phase values]
```

**Compressed (compressionType=1):**

```text
[int32: levelCount]
[int32: compressedLevelLen]
[byte * compressedLevelLen * 2: compressed level data]
[int32: phaseCount]
[int32: compressedPhaseLen]
[byte * compressedPhaseLen * 2: compressed phase data]
```

### Version 1: TransferFunctionLsPs (Current Format)

Uses ComplexSpectrum structure with nested ComplexSequence.

| Field         | Type                  | Description               |
| ------------- | --------------------- | ------------------------- |
| block_size    | int32                 | Total block size          |
| version_check | int16                 | Must be 0                 |
| sub_version   | int16                 | Feature version           |
| Definition    | LogSpectrumDefinition | Frequency grid (16 bytes) |
| Sequence      | ComplexSequence       | Level and phase data      |
| Delay         | double                | Group delay in seconds    |

---

## ComplexSequence

Container for two Record arrays (level and phase data).

| Field         | Type   | Description      |
| ------------- | ------ | ---------------- |
| block_size    | int32  | Total block size |
| version_check | int16  | Must be 0        |
| sub_version   | int16  | Feature version  |
| Record[0]     | Record | Level data       |
| Record[1]     | Record | Phase data       |

---

## Record (Compressed Array)

Stores an array of int16 values with optional BitCompression.

| Field           | Type   | Description             |
| --------------- | ------ | ----------------------- |
| block_size      | int32  | Total block size        |
| version_check   | int16  | Must be 0               |
| sub_version     | int16  | Feature version         |
| compressionType | int32  | 0=raw, 1=BitCompression |
| elementCount    | int32  | Number of values        |
| data            | varies | Raw or compressed data  |

**Uncompressed (compressionType=0):**

```text
[int16 * elementCount: raw values]
```

**Compressed (compressionType=1):**

```text
[int32: compressedLen]
[byte * compressedLen * 2: BitCompression encoded data]
```

---

## BitCompression Algorithm

A variable bit-depth compression algorithm used by EASE/AFMG for spectrum data.

### Encoding

Data is compressed in groups of 8 values (fixed step size). Each group has:

1. **4-bit header**: Contains `(bit_depth - 1)` where bit_depth is 1-16
2. **N values**: Each value uses `bit_depth` bits

Values are stored as signed integers using sign extension.

### Decoding

```text
bit_pos = 0
for each group of 8 values:
    bit_depth = read_bits(4) + 1
    for i in 0..7:
        raw_value = read_bits(bit_depth)
        values[i] = sign_extend(raw_value, bit_depth)
```

### Delta Encoding (Differentiation)

When the `differentiated` flag is set, values are stored as deltas:

- First value is absolute
- Subsequent values are differences from the previous value

To decode: integrate (cumulative sum) after decompression.

### Value Scaling

After decompression:

- **Level values**: multiply by 0.01 to get dB
- **Phase values**: multiply by 0.001 to get radians

---

## Limit

Defines mechanical or electrical limits for array configurations.

| Field      | Type   | Description                  |
| ---------- | ------ | ---------------------------- |
| block_size | int32  | Total block size             |
| vcheck     | int16  | Must be 0                    |
| sver       | int16  | Feature version              |
| Frame      | string | Frame key this applies to    |
| Type       | int32  | Limit type (see below)       |
| BoxType    | string | Box type key this applies to |
| LimitValue | double | The limit value              |

### Limit Types

| Value | Type         | Description                |
| ----- | ------------ | -------------------------- |
| 0     | MaxCount     | Maximum array count        |
| 1     | MaxCountType | Maximum count per box type |
| 2     | MaxWeightKg  | Maximum weight in kg       |
| 4     | MaxTiltAngle | Maximum tilt angle (deg)   |
| 5     | MinTiltAngle | Minimum tilt angle (deg)   |
| 6     | MinCount     | Minimum array count        |

---

## Warning

Defines configuration warning messages.

| Field      | Type   | Description                       |
| ---------- | ------ | --------------------------------- |
| block_size | int32  | Total block size                  |
| vcheck     | int16  | Must be 0                         |
| sver       | int16  | Feature version                   |
| Frame      | string | Frame key this applies to         |
| Type       | int32  | Warning type (see below)          |
| Text       | string | Warning message text              |
| LimitValue | double | Limit value that triggers warning |

### Warning Types

| Value | Type         | Description                |
| ----- | ------------ | -------------------------- |
| 0     | MaxCount     | Maximum count warning      |
| 1     | MinCount     | Minimum count warning      |
| 2     | MaxWeightKg  | Maximum weight warning     |
| 3     | MaxTiltAngle | Maximum tilt angle warning |
| 4     | MinTiltAngle | Minimum tilt angle warning |

---

## FilterGroup

Contains DSP filter definitions for crossovers and EQ.

| Field         | Type                   | Description                   |
| ------------- | ---------------------- | ----------------------------- |
| block_size    | int32                  | Total block size              |
| vcheck        | int16                  | Must be 0                     |
| sver          | int16                  | Feature version               |
| Label         | string                 | Display name                  |
| Key           | string                 | Group identifier              |
| FilterDefs    | FilterDefinitionBuffer | Nested buffer of filter defs  |
| IsOverridable | int16                  | Can be overridden (sver >= 1) |

### FilterDefinition

| Field      | Type              | Description           |
| ---------- | ----------------- | --------------------- |
| block_size | int32             | Total block size      |
| vcheck     | int16             | Must be 0             |
| sver       | int16             | Feature version       |
| Label      | string            | Display name          |
| Key        | string            | Filter identifier     |
| Filter     | GenericFilterBank | Filter bank structure |

---

## GenericFilterBank

Container for DSP filter chains with global settings.

| Field          | Type                    | Description                   |
| -------------- | ----------------------- | ----------------------------- |
| block_size     | int32                   | Total block size              |
| vcheck         | int16                   | Must be 0                     |
| sver           | int16                   | Feature version (0 or 1)      |
| reserved       | int32                   | Always 0                      |
| ByPass         | int32                   | 1=bypass filter chain         |
| InvertPolarity | int32                   | 1=invert polarity (180° flip) |
| Gain           | double                  | Overall gain in dB            |
| Delay          | double                  | Overall delay in seconds      |
| Filters        | GenericBaseFilterBuffer | Chain of individual filters   |
| MuteInput      | int32                   | 1=mute input (sver >= 1 only) |

### GenericBaseFilterBuffer

| Field      | Type        | Description              |
| ---------- | ----------- | ------------------------ |
| block_size | int32       | Total block size         |
| vcheck     | int16       | Must be 0                |
| sver       | int16       | Feature version          |
| count      | int32       | Number of filters        |
| filters[]  | (see below) | Individual filter blocks |

For each filter:

1. Read `FilterKind` (int32): 0=LogSpectrum, 1=IIR, 2=FIR
2. Read filter block based on kind

---

## GenericBaseFilter (Base Class)

All filter types share these base fields (read via `LoadFromSourceBase`):

| Field          | Type   | Description             |
| -------------- | ------ | ----------------------- |
| block_size     | int32  | Total block size        |
| vcheck         | int16  | Must be 0               |
| sver           | int16  | Feature version         |
| reserved       | int32  | Always 0                |
| ByPass         | int32  | 1=bypass this filter    |
| InvertPolarity | int32  | 1=invert polarity       |
| Gain           | double | Filter gain in dB       |
| Delay          | double | Filter delay in seconds |
| Label          | string | Display name            |
| Key            | string | Filter identifier       |

---

## LogSpectrumFilter (FilterKind=0)

Defines a filter using a stored frequency response curve.

| Field          | Type               | Description                          |
| -------------- | ------------------ | ------------------------------------ |
| block_size     | int32              | Total block size                     |
| vcheck         | int16              | 0 or 1 (affects spectrum format)     |
| sver           | int16              | Feature version                      |
| (base fields)  | GenericBaseFilter  | See base class fields above          |
| RawLogSpectrum | TransferFunctionLP | Level/phase response (format varies) |

**Format selection:**

- `vcheck=0`: Reads `CLogSpectrumLP` (older format), converts internally
- `vcheck=1`: Reads `TransferFunctionLsPs` directly

---

## IIRFilter (FilterKind=1)

Defines an IIR (Infinite Impulse Response) biquad filter.

| Field          | Type              | Description                 |
| -------------- | ----------------- | --------------------------- |
| block_size     | int32             | Total block size            |
| vcheck         | int16             | Must be 0                   |
| sver           | int16             | Feature version             |
| (base fields)  | GenericBaseFilter | See base class fields above |
| FilterFunction | FilterFunction    | IIR filter parameters       |

### FilterFunction (IIR Parameters)

| Field              | Type   | Description                      |
| ------------------ | ------ | -------------------------------- |
| block_size         | int32  | Total block size                 |
| vcheck             | int16  | Must be 0                        |
| sver               | int16  | Feature version                  |
| FilterType         | int32  | Filter type enum (see below)     |
| FilterShape        | int32  | Filter shape enum (see below)    |
| Order              | int32  | Filter order (1-8)               |
| FreqCritInHz       | double | Critical/center frequency (Hz)   |
| Alignment          | int32  | Frequency alignment enum         |
| reserved           | double | Always 0.0                       |
| QFactor            | double | Q factor (for SallenKey shape)   |
| ParametricGainIndB | double | Gain for peak/shelf filters (dB) |

### FilterType Enum

| Value | Name      | Description          |
| ----- | --------- | -------------------- |
| 0     | LowPass   | Low-pass filter      |
| 1     | HighPass  | High-pass filter     |
| 2     | AllPass   | All-pass filter      |
| 3     | Peak      | Parametric peak EQ   |
| 4     | PeakSym   | Symmetric peak EQ    |
| 5     | LowShelf  | Low shelving filter  |
| 6     | HighShelf | High shelving filter |

### FilterShape Enum

| Value | Name          | Description                       |
| ----- | ------------- | --------------------------------- |
| 0     | Butterworth   | Maximally flat magnitude          |
| 1     | LinkwitzRiley | Linkwitz-Riley (even orders only) |
| 2     | Bessel        | Maximally flat group delay        |
| 3     | SallenKey     | 2nd-order with adjustable Q       |

### FilterAlign Enum

| Value | Name         | Description                      |
| ----- | ------------ | -------------------------------- |
| 0     | None         | No alignment                     |
| 1     | Level3dB     | -3dB at critical frequency       |
| 2     | Level6dB     | -6dB at critical frequency       |
| 3     | PhaseMatched | Phase-matched alignment (Bessel) |

---

## FIRFilter (FilterKind=2)

Defines a FIR (Finite Impulse Response) filter.

| Field         | Type              | Description                   |
| ------------- | ----------------- | ----------------------------- |
| block_size    | int32             | Total block size              |
| vcheck        | int16             | Must be 0                     |
| sver          | int16             | Feature version               |
| (base fields) | GenericBaseFilter | See base class fields above   |
| LinResponse   | CLinResponse      | Time or frequency domain data |

### CLinResponse (FIR Data)

| Field      | Type     | Description                                      |
| ---------- | -------- | ------------------------------------------------ |
| block_size | int32    | Total block size                                 |
| vcheck     | int16    | Must be 0                                        |
| sver       | int16    | Feature version                                  |
| flags      | int32    | Bit0=isTimeResponse, Bit1=isComplex, Bit2=isEven |
| dataIRMLen | int32    | Length of IRM data array                         |
| dataIRM[]  | double[] | Real part (time) or magnitude (frequency)        |
| dataDIPLen | int32    | Length of DIP data array                         |
| dataDIP[]  | double[] | Imaginary part (time) or phase (frequency)       |
| sampleRate | double   | Sample rate in Hz (e.g., 48000.0)                |

**Data interpretation:**

- `isTimeResponse=1`: dataIRM is time-domain impulse response
- `isTimeResponse=0`: dataIRM/dataDIP are frequency-domain real/imag
- `isComplex=1`: Complex data (both arrays used)
- `isEven=1`: Even-symmetric for FFT optimization

---

## ClusterSetupItem

Wrapper for predefined array configurations.

| Field      | Type         | Description            |
| ---------- | ------------ | ---------------------- |
| block_size | int32        | Total block size       |
| vcheck     | int16        | 0 or 1 (format varies) |
| sver       | int16        | Feature version        |
| Label      | string       | Display name           |
| Key        | string       | Setup identifier       |
| Setup      | ClusterSetup | Array configuration    |

**Note:** When vcheck=0, the Setup is just a ClusterBoxBuffer directly. When vcheck=1, the full ClusterSetup structure is used.

### ClusterSetup

| Field       | Type             | Description                |
| ----------- | ---------------- | -------------------------- |
| block_size  | int32            | Total block size           |
| vcheck      | int16            | Must be 0                  |
| sver        | int16            | Feature version            |
| Description | string           | Configuration description  |
| Boxes       | ClusterBoxBuffer | Speaker cabinet placements |

### ClusterBox

Represents a speaker cabinet in a cluster configuration.

| Field      | Type      | Description                           |
| ---------- | --------- | ------------------------------------- |
| block_size | int32     | Total block size                      |
| vcheck     | int16     | Must be 0                             |
| sver       | int16     | Feature version (0 or 1)              |
| reserved1  | int32     | Reserved (always 0)                   |
| reserved2  | int32     | Reserved (always 0)                   |
| HashID     | int32     | Unique identifier for this box        |
| Label      | string    | Display name                          |
| reserved3  | double    | Reserved (always 0.0)                 |
| BoxTypeKey | string    | Key of box type to use                |
| [padding]  | int32 × 4 | Only if sver=0 (legacy compatibility) |
| Position   | Vector3D  | X, Y, Z position (3 doubles, no wrap) |
| [padding]  | int32 × 2 | Only if sver=0 (legacy compatibility) |
| Angles     | Vector3D  | H, V, R rotation (3 doubles, radians) |

**Note:** Vector3D here is NOT wrapped in a block - it's just 3 consecutive doubles (24 bytes).

---

## GenSystemPreset

System preset configuration (Database sver >= 1).

| Field      | Type            | Description             |
| ---------- | --------------- | ----------------------- |
| block_size | int32           | Total block size        |
| vcheck     | int16           | Must be 0               |
| sver       | int16           | Feature version         |
| Label      | string          | Display name            |
| Key        | string          | Preset identifier       |
| Config     | GenSystemConfig | Complex config (parsed) |

The GenSystemConfig structure is complex and contains element configurations, grid angles, and cluster setup references. For most use cases, only Label and Key are needed.

---

## IncludeFile

Additional embedded file (Database sver >= 2). Similar to DataFile but includes label and key.

| Field      | Type   | Description       |
| ---------- | ------ | ----------------- |
| block_size | int32  | Total block size  |
| vcheck     | int16  | Must be 0         |
| sver       | int16  | Feature version   |
| Label      | string | Display name      |
| Key        | string | File identifier   |
| Filename   | string | Original filename |
| Size       | int32  | Byte count        |
| Bytes      | byte[] | File content      |

Common uses:

- PDF documentation files
- Additional configuration files

---

## AuthorFile

Authorization/licensing data (Database sver >= 2).

This buffer uses the same format as DataFileBuffer. The .author files contain encrypted licensing and authorization information for protected GLL files.

---

## Transformer

Power transformer configuration (Database sver >= 3).

| Field         | Type             | Description                   |
| ------------- | ---------------- | ----------------------------- |
| block_size    | int32            | Total block size              |
| vcheck        | int16            | Must be 0                     |
| sver          | int16            | Feature version               |
| Label         | string           | Display name                  |
| Key           | string           | Transformer identifier        |
| MaxPower      | double           | Maximum power in watts        |
| NetVoltage    | double           | Network voltage (e.g., 70.7V) |
| LspkImpedance | double           | Loudspeaker impedance (ohms)  |
| TapSettings   | TapSettingBuffer | Available tap configurations  |

### TapSetting

Defines a transformer tap configuration.

| Field      | Type   | Description           |
| ---------- | ------ | --------------------- |
| block_size | int32  | Total block size      |
| vcheck     | int16  | Must be 0             |
| sver       | int16  | Feature version       |
| Label      | string | Tap label             |
| Key        | string | Tap identifier        |
| PowerRatio | double | Power ratio (0.0-1.0) |

---

## DataFile

Embedded binary file.

| Field    | Type   | Description       |
| -------- | ------ | ----------------- |
| Key      | string | File identifier   |
| Filename | string | Original filename |
| Bytes    | byte[] | File content      |

Common embedded files:

- PNG images (logos, photos)
- 3D geometry (.xed)
- Configuration files

---

## Vector3D

3D coordinate.

| Field | Type   | Description      |
| ----- | ------ | ---------------- |
| X     | double | X coordinate (m) |
| Y     | double | Y coordinate (m) |
| Z     | double | Z coordinate (m) |

---

## CaseGeometry (3D Mesh Data)

Contains 3D mesh data for speaker cabinet and frame geometry. Used for visual representation and acoustic calculations.

### CaseGeometry Structure

| Field        | Type         | Description                          |
| ------------ | ------------ | ------------------------------------ |
| block_size   | int32        | Total block size                     |
| vcheck       | int16        | Must be 0                            |
| sver         | int16        | Sub-version (0 or 1)                 |
| IsSymmetric  | int32        | 0=no, 1=yes (planar symmetry on X)   |
| SymmetryAxis | double       | X coordinate of symmetry plane       |
| Vertices     | VertexBuffer | 3D vertex positions                  |
| Edges        | EdgeBuffer   | Vertex index pairs forming edges     |
| Faces        | FaceBuffer   | Face definitions (only if sver >= 1) |

**Symmetry:** When IsSymmetric=1, geometry is mirrored across the plane X=SymmetryAxis. Negative vertex indices in edges/faces refer to mirrored vertices.

### Vertex

Each vertex has a block wrapper and metadata.

| Field      | Type   | Description                    |
| ---------- | ------ | ------------------------------ |
| block_size | int32  | Total block size               |
| vcheck     | int16  | Must be 0                      |
| sver       | int16  | Sub-version                    |
| Color      | int32  | RGB color for rendering        |
| X          | double | X coordinate (m)               |
| Y          | double | Y coordinate (m)               |
| Z          | double | Z coordinate (m)               |
| Label      | string | Optional vertex label          |
| HasTwin    | byte   | 1 if vertex has symmetric twin |

**Note:** Coordinates are stored as `double` (8 bytes each), not `float32`.

### Edge

Each edge has a block wrapper and metadata.

| Field      | Type   | Description                               |
| ---------- | ------ | ----------------------------------------- |
| block_size | int32  | Total block size                          |
| vcheck     | int16  | Must be 0                                 |
| sver       | int16  | Sub-version                               |
| Color      | int32  | RGB color for rendering                   |
| V1         | int32  | First vertex index (negative = mirrored)  |
| V2         | int32  | Second vertex index (negative = mirrored) |
| Label      | string | Optional edge label                       |
| HasTwin    | byte   | 1 if edge has symmetric twin              |

### Face

Each face has a block wrapper and variable vertex count.

| Field       | Type    | Description                          |
| ----------- | ------- | ------------------------------------ |
| block_size  | int32   | Total block size                     |
| vcheck      | int16   | Must be 0                            |
| sver        | int16   | Sub-version                          |
| HasTwin     | byte    | 1 if face has symmetric twin         |
| VertexCount | int32   | Number of vertices in this face      |
| Vertices    | int32[] | Vertex indices (negative = mirrored) |
| Color       | int32   | RGB color for rendering              |
| Label       | string  | Optional face label                  |

### Buffer Wrappers

VertexBuffer, EdgeBuffer, and FaceBuffer each use the standard ObjectLSCBuffer pattern:

| Field      | Type   | Description            |
| ---------- | ------ | ---------------------- |
| block_size | int32  | Total buffer size      |
| vcheck     | int16  | Must be 0              |
| sver       | int16  | Sub-version            |
| count      | int32  | Number of items        |
| items[]    | Item[] | Individual item blocks |

### Related File Formats

Geometry can also be stored in external files:

| Extension | Format     | Description              |
| --------- | ---------- | ------------------------ |
| `.xed`    | ASCII Edge | Text-based edge geometry |
| `.fed`    | EASE Edge  | Binary EASE edge format  |
| `.xfc`    | ASCII Face | Text-based face geometry |
| `.ffc`    | EASE Face  | Binary EASE face format  |

These files may be embedded in the DataFiles buffer or referenced externally.

---

## Checksum Algorithm

The file integrity checksum (version >= 4):

```python
def calculate_checksum(data, start, end):
    checksum = [0, 0, 0, 0]
    for i in range(start, end):
        b = data[i]
        checksum[0] = (b ^ (123 + checksum[0])) % 256
        checksum[1] = ((11 * b) ^ (1433 + checksum[1])) % 256
        checksum[2] = ((3 * b) ^ (13 + checksum[2])) % 256
    checksum[3] = ((checksum[0] + checksum[1] + checksum[2]) ^ 0x37) % 256
    return checksum
```

---

## Related File Types

| Extension        | Description                                  |
| ---------------- | -------------------------------------------- |
| `.gll`           | Compiled GLL file (this format)              |
| `.xgll`          | Text-based GLL project (source format)       |
| `.gss`           | Generic Sound Source data (embedded in .gll) |
| `.gfb` / `.xgfb` | Generic Filter Bank                          |
| `.xed`           | EASE 3D geometry                             |
| `.frd`           | Frequency Response Data                      |
