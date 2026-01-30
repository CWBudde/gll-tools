# CLF Text Format Specification (Common Loudspeaker Format)

This document describes the CLF text format used for loudspeaker directivity data exchange.
The specification was reverse-engineered from existing implementations.

## Overview

CLF is an industry-standard format for loudspeaker directivity data, maintained by the
[CLF Group](http://www.clfgroup.org/). It exists in two parts:

- **Text format** (CT1/CT2): TAB-delimited text files for authoring and editing
- **Binary format** (CF1/CF2): Secure binary distribution files created by the CLF Editor

This project targets the **text format** only. Users can convert text files to binary
CF1/CF2 using the official CLF Editor/Viewer.

## Format Versions

| Feature                         | CLF1 (Type 1)    | CLF2 (Type 2)         |
| ------------------------------- | ---------------- | --------------------- |
| Angular resolution              | 10°              | 5°                    |
| Frequency bands                 | Octave (9 bands) | 1/3-octave (24 bands) |
| Azimuth values (full sphere)    | 36               | 72                    |
| Azimuth values (half sphere)    | 19               | 37                    |
| Azimuth values (quarter sphere) | 10               | 19                    |

CLF2 (Type 2) matches the GLL balloon grid resolution (5°, 72×37 points).

## Text File Structure

The file is TAB-delimited. Each line starts with a keyword token followed by TAB-separated values.
The file is structured in two parts: a header section and a data section.

### File Start

```
<CLF1>           or <CLF2>           # Format type identifier
```

### Header Fields

All header fields use the format: `<KEYWORD>` TAB `value`

```
<VERSION>                <int>         # Format version (typically 1)
<MODELNAME>              <string>      # Speaker model name
<INFOFILE>               <string>      # Info file reference (optional)
<MANUFACTURER>           <string>      # Manufacturer name
<WEB-SITE>               <string>      # Website URL
<DESCRIPTION>            <string>      # Product description
<COLORS>                 <string>      # Cabinet color options
<MOUNTING>               <string>      # Mounting information
<WEIGHT>                 <float>       # Weight in kilograms
<MINBAND>                <float>       # Minimum frequency band (Hz)
<MAXBAND>                <float>       # Maximum frequency band (Hz)
<MEASUREMENT-CONTACT>    <string>      # Contact person for measurements
<MEASUREMENT-EMAIL>      <string>      # Contact email
<MEASUREMENT-DATE>       <string>      # Date in YYYY-MMM-DD format (e.g. 2024-JAN-31)
<MEASUREMENT-NOTE>       <string>      # Measurement notes
<MEASUREMENT-ENVIRONMENT> <string>     # Measurement environment description
<MEASUREMENT-DISTANCE>   <float>       # Measurement distance in meters (typically 1)
<MEASUREMENT-INPUTVOLTAGE> <string>    # Test voltage range
<TYPE>                   <string>      # "passive", "active", or "powered"
<SENSITIVITY>            <values>      # TAB-separated dB sensitivity per frequency band (optional)
<SENSITIVITY-INFO>       <string>      # Sensitivity measurement method/notes
<IMPEDANCE>              <float>       # Nominal impedance in ohms
<IMPEDANCE-INFO>         <string>      # Impedance measurement notes
<TOTMAXINPUT>            <voltage> <method>  # Maximum input and method
<RADIATION>              <string>      # "halfsphere" or "fullsphere"
<AXIAL-SPECTRUM>         <values>      # TAB-separated on-axis frequency response (optional)
<AXIAL-SPECTRUM-INFO>    <string>      # Axial spectrum notes (e.g. "at 1m")
<BALLOON-SYMMETRY>       <type>        # Symmetry type (see below)
<BALLOON-ARC-ORDER>      <string>      # Azimuth starting angle and rotation direction
<BALLOON-REF>            <string>      # "absolute" or "relative"
```

### Balloon Symmetry Types

| Value          | Description                    | Azimuth values (CLF1) | Azimuth values (CLF2) |
| -------------- | ------------------------------ | --------------------- | --------------------- |
| `<none>`       | Full sphere, no symmetry       | 36                    | 72                    |
| `<horizontal>` | Horizontal plane symmetry      | 19                    | 37                    |
| `<full>`       | Full (quarter sphere) symmetry | 10                    | 19                    |
| `<rotational>` | Rotational symmetry            | 1                     | 1                     |
| `<polar>`      | Polar symmetry (90° intervals) | 4                     | 4                     |
| `<vertical>`   | Vertical plane symmetry        | 1                     | 1                     |

### Data Section

After the header, directivity data is organized by frequency band:

```
<BAND>    <frequency_hz>
<value>   <value>   ...   <value>     # One row per azimuth angle
<value>   <value>   ...   <value>     # Values are TAB-separated polar levels (dB)
...                                    # Number of columns = number of polar angles
<BAND>    <frequency_hz>               # Next frequency band
...
```

Data organization:

- Each `<BAND>` block contains one frequency
- Each row within a band corresponds to one azimuth angle
- Each column corresponds to one polar (elevation) angle
- Values are SPL levels in dB

### File End

```
<CABINET-SYSTEM>          # Optional: cabinet geometry information
<CLF1END>   or <CLF2END>  # File terminator
```

## Frequency Bands

### CLF1 (Octave bands, 9 bands)

125, 250, 500, 1000, 2000, 4000, 8000, 16000 Hz (standard octave centers)

### CLF2 (1/3-octave bands, 24 bands)

100, 125, 160, 200, 250, 315, 400, 500, 630, 800, 1000, 1250, 1600, 2000,
2500, 3150, 4000, 5000, 6300, 8000, 10000, 12500, 16000, 20000 Hz

## GLL to CLF Mapping

| GLL Field                    | CLF Field                        |
| ---------------------------- | -------------------------------- |
| GenSystem.Manufacturer       | `<MANUFACTURER>`                 |
| GenSystem.DeviceName         | `<MODELNAME>`                    |
| GenSystem.Description        | `<DESCRIPTION>`                  |
| GenSystem.WebLink            | `<WEB-SITE>`                     |
| BalloonData (72×37 grid, 5°) | CLF2 `<BAND>` data blocks        |
| SourceDefinition responses   | Resampled to 1/3-octave for CLF2 |
| BalloonData.SymmetryType     | `<BALLOON-SYMMETRY>`             |

### Angular Grid Mapping

GLL uses:

- Meridian: 0-360° in 5° steps (72 points) → CLF azimuth
- Parallel: 0-180° in 5° steps (37 points) → CLF polar

CLF2 uses the same 5° resolution, so the grid maps directly.
The `<BALLOON-ARC-ORDER>` field controls the starting azimuth and rotation direction,
which may need adjustment depending on GLL's coordinate convention.

### Frequency Resampling

GLL transfer functions contain high-resolution frequency data that must be
resampled/interpolated to the standard 1/3-octave center frequencies for CLF2 output.
The nearest frequency bin or energy-averaged band level should be used.

## References

- [CLF Group](http://www.clfgroup.org/) — Official CLF format maintainer
- [Pro Sound Training: What is CLF?](https://www.prosoundtraining.com/2010/03/06/clf-news-what-is-the-clf/)
- MonkeySphere legacy code (MBFormats.pas, MBMain.pas) — Reference implementation
