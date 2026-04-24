# Acoustic Model Notes

This document captures the current acoustic assumptions used by the visualization and array-response code. It is not a replacement for the binary format specification; it is the implementation contract that needs reference validation against EASE GLL Viewer output.

## Coordinate System

- Positions passed to the Go array-response engine are in meters.
- Parsed GLL geometry and source placement coordinates are stored in millimeters and converted by UI/API adapters before calculation.
- The acoustic engine treats `+X` as the local firing axis:
  - `parallel=0 deg`: front/on-axis (`+X`)
  - `parallel=180 deg`: rear (`-X`)
  - `meridian=0 deg`: top (`+Z`)
  - `meridian=90 deg`: right (`+Y`)
- Source placement rotations are H/V/R angles in radians. The web demo passes an explicit world-from-local orientation matrix for source placements; the Go engine falls back to Euler angles when no matrix is supplied.

## Source Response Components

Each `SourceDefinition` can contain two relevant acoustic response components:

- `BalloonData.Responses`: transfer functions sampled on the angular directivity grid.
- `OnAxisSpectrum`: separately parsed on-axis frequency response.

Current visualization behavior:

- Single-source response charts and polar slices treat balloon responses as directivity data and optionally add `OnAxisSpectrum` when the frequency grids match.
- The array-response engine currently multiplies an interpolated balloon response by the balloon response at `(meridian=0, parallel=0)`.

Open contract to validate:

- Whether `BalloonData.Responses` are relative directivity-only data, absolute SPL data, or data requiring a separate on-axis correction.
- Whether array response should use `SourceDefinition.OnAxisSpectrum` instead of the front-pole balloon response for on-axis normalization.
- Whether `OnAxisLevel` should be folded into chart labels, SPL calibration, or future sensitivity export.

## Phase And Delay

Transfer functions are stored as level in dB and phase in radians. Coherent summation converts level/phase to complex pressure, sums complex values, and converts back to level/phase.

Current implementation behavior:

- `TransferFunction.AddDelay` adds `+2*pi*f*delay` to phase.
- Response visualization subtracts `2*pi*f*delay` before unwrapping and computing group delay.
- The existing unit test suite characterizes the positive phase-addition behavior.

Open contract to validate:

- Confirm the sign convention used by parsed GLL phase data.
- Decide whether propagation delay should be represented as positive stored phase, negative display phase, or normalized before summation.
- Add a two-source synthetic interference test with known null/peak frequencies before changing this behavior.

## Angular Interpolation

Current implementation behavior:

- Angular samples are interpolated as complex pressure:
  - `level` is converted from dB to linear pressure magnitude.
  - `phase` is applied as the complex angle.
  - The surrounding angular samples are weighted in the complex plane.
  - Interpolated `level` and `phase` are derived from the resulting complex value.
- The Go array-response engine and the web demo's display-only single-source
  polar/balloon paths use the same complex-pressure interpolation contract.

Open contract to validate:

- Validate complex-pressure interpolation against EASE GLL Viewer output.

## Air Attenuation

Current implementation behavior:

- Air attenuation is optional in the array-response engine.
- The attenuation coefficient is calculated from the ISO 9613-1 atmospheric absorption model using frequency, temperature, relative humidity, and pressure.
- The web demo exposes temperature, humidity, and pressure inputs for array-response and array-balloon calculations.
- Temperature still does not automatically change propagation speed; speed remains an explicit air property with the existing 343 m/s fallback.

Open contract to validate:

- Validate ISO 9613-1 reference points against an independent implementation or published calculator.
- Decide whether the web demo should expose propagation speed separately from attenuation temperature.
