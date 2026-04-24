# Validation Workflow

This document tracks reference comparisons for visualization output. Reference
data must come from official EASE GLL Viewer exports or screenshots; do not
mark a case validated from parser output alone.

## Reference Cases

| Case              | GLL file | Reference source                   | Status  | Notes                                                           |
| ----------------- | -------- | ---------------------------------- | ------- | --------------------------------------------------------------- |
| Small loudspeaker | TBD      | EASE GLL Viewer export/screenshots | Pending | Capture on-axis, off-axis, polar, and balloon data.             |
| Line array        | TBD      | EASE GLL Viewer export/screenshots | Pending | Capture combined array response for a documented configuration. |

## Capture Checklist

For each reference case, store enough information to reproduce the comparison:

- GLL filename, file hash, and source/vendor version.
- Viewer version and operating system used for export.
- Receiver position, array configuration, active preset/filter state, and air
  attenuation settings.
- On-axis response CSV or screenshot.
- Off-axis response at a fixed documented angle.
- Horizontal and vertical polar slices at 1 kHz, 4 kHz, and 8 kHz when data is
  available.
- Combined array response for at least one multi-element configuration.

## Tolerances

Initial comparison tolerances before reference-specific adjustments:

- Frequency bins: exact match when the exported grids are identical; otherwise
  compare nearest log-frequency bins and record the bin mismatch.
- Level: `<= 0.5 dB` for parsed single-source samples, `<= 1.0 dB` for combined
  array samples until air/path assumptions are fully validated.
- Phase: `<= 0.05 rad` after applying the documented unwrap convention.
- Polar angle samples: `<= 0.5 dB` at measured grid points; interpolated points
  must be identified separately.

## Discrepancy Log

| Date | Case | Surface | Difference | Likely cause | Resolution |
| ---- | ---- | ------- | ---------- | ------------ | ---------- |
| TBD  | TBD  | TBD     | TBD        | TBD          | TBD        |

## Synthetic Fixture Coverage

The browser-facing array computation core has deterministic Go tests in
`cmd/gllwasm/compute_core_test.go`. The fixture is intentionally tiny:

- One source definition with a 90 degree balloon grid.
- Known levels for front, right, top, back, bottom, and left directions.
- A one-band 1 kHz transfer function with zero phase.

The tests verify:

- `computeArrayResponse` core output for a known receiver direction.
- `computeArrayBalloon` core progress callbacks.
- Final balloon result parity against repeated single-receiver response calls.

These tests protect API regressions but are not a substitute for EASE Viewer
reference validation.
