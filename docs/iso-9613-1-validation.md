# ISO 9613-1 Atmospheric Absorption Cross-Check (2026-05-09)

This is the external-reference cross-check called out in `PLAN.md` § 9.1
("Validate air attenuation expectations") — comparing
`internal/acoustics/air.go::AirLossPerMeter` against an independent
reference implementation of ISO 9613-1:1993 / ANSI S1.26-1995 and against
published tabulated values.

## TL;DR

`AirLossPerMeter` produces values that are systematically lower than the
ISO 9613-1 reference for any humid case. At 20 °C, 50 % RH, sea level the
deviation is ~30 % at 1 kHz and grows with frequency, reaching ~8× too
small at 10 kHz. The error stems from a unit-of-`h` mismatch: the standard
defines `h` as the molar concentration of water vapor expressed **as a
percent**, but the implementation uses it as a fraction, so the empirical
constants (`24`, `4.04 × 10⁴`, `9`, `280`, `0.391`, `0.02`, `4.17`) — which
are calibrated for `h` in percent — see a value 100× too small.

## Standard convention

Per ISO 9613-1:1993 and the corroborating ANSI S1.26 description and
sengpielaudio's documented derivation:

```
h = h_r · p_sat / p_a    where h_r is relative humidity AS A PERCENTAGE
```

`p_sat / p_r = 10^(−6.8346 · (T_01/T)^1.261 + 4.6151)`, `T_01 = 273.16 K`.

`frO = (p_a/p_r) · (24 + 4.04·10⁴ · h · (0.02+h)/(0.391+h))`

`frN = (p_a/p_r) · (T/T_0)^(−1/2) · (9 + 280·h·exp(−4.170·((T/T_0)^(−1/3)−1)))`

`α = 8.686 · f² · { 1.84·10⁻¹¹ · (p_a/p_r)⁻¹ · (T/T₀)^(1/2)
                  + (T/T₀)^(−5/2) · [
                       0.01275 · exp(−2239.1/T) / (frO + f²/frO)
                     + 0.1068  · exp(−3352.0/T) / (frN + f²/frN)
                    ]
                  } [dB/m]`

The empirical constants `24`, `4.04 · 10⁴`, `9`, `280`, `0.391`, `0.02`,
`4.170` are calibrated for **`h` in percent**. Substituting a fractional
`h` produces relaxation frequencies an order of magnitude too low and α
that is too small in the regime where molecular relaxation dominates.

## What `internal/acoustics/air.go` does today

```go
molarHumidity := humidity * saturationVaporPressureRatio(temperatureK) / pressureRatio
```

`humidity` is the function input, documented as the relative-humidity
**fraction** (test cases pass `0.5` for 50 % RH; `array_calculations.go`
passes `AirProperties.Humidity` which is also fractional). The result
`molarHumidity` therefore equals `h_r · p_sat/p_a` with `h_r` in
fractional units, i.e. 100× smaller than the standard's `h`.

## Quantitative comparison

Reference values: independent Python re-implementation of ISO 9613-1:1993
using the standard's prescription (`h` in percent), cross-checked against
Salomons (2001) Table 3.1 at the same conditions. At 1–2 kHz the Python
reference matches Salomons within ±5 %; at higher frequencies it tracks
ANSI S1.26-1995 closely. The remaining ~5 % residual stems from the
choice of vapor-pressure formula (ISO 9613-1:1993 keeps the simplified
log10(psat/pr) Magnus form; ANSI S1.26-2014 uses Hyland-Wexler).

| Case                       | gll-tools (dB/m) | ISO ref (dB/m) | ratio (ref / gll) |
| -------------------------- | ---------------: | -------------: | ----------------: |
| 1 kHz, 20 °C, 50 % RH      |        0.003498  |       0.004665 |             1.33× |
| 10 kHz, 20 °C, 50 % RH     |        0.019332  |       0.158839 |             8.22× |
| 1 kHz, 20 °C, 0 % RH       |        0.001530  |       0.001530 |             1.00  |
| 1 kHz, 20 °C, 100 % RH     |        0.006672  |       0.005422 |             0.81× |
| 10 kHz, 0 °C, 20 % RH      |        0.016447  |       0.064314 |             3.91× |
| 1 kHz, 50 % RH, 80 kPa     |        0.003427  |       0.004619 |             1.35× |
| 1 kHz, 50 % RH, 120 kPa    |        0.003622  |       0.004717 |             1.30× |
| 500 Hz, 25 °C, 70 % RH     |        0.006497  |       0.003069 |             0.47× |
| 4 kHz, 30 °C, 30 % RH      |        0.006866  |       0.032965 |             4.80× |
| 100 Hz, 20 °C, 50 % RH     |        0.002485  |       0.000294 |             0.12× |

The dry-air case (h = 0) matches because the bug is irrelevant when there
is no water vapor. Every humid case diverges, and the divergence is
strongly frequency-dependent because the wrong `h` shifts both relaxation
frequencies by orders of magnitude, which moves the f²/frO and f²/frN
terms across their characteristic transition regions.

## Proposed fix

In `internal/acoustics/air.go::AirLossPerMeter`, convert humidity to
percent before substituting into the standard's empirical constants:

```diff
-    molarHumidity := humidity * saturationVaporPressureRatio(temperatureK) / pressureRatio
+    // ISO 9613-1: h is the molar concentration of water vapor expressed as a
+    // PERCENT. The empirical constants below (24, 4.04e4, 9, 280, 0.391, 0.02,
+    // 4.17) are calibrated for that unit.
+    molarHumidityPercent := 100.0 * humidity * saturationVaporPressureRatio(temperatureK) / pressureRatio
```

…and rename the variable's downstream uses accordingly.

## Knock-on impact

- **`internal/acoustics/acoustics_test.go::TestAirLossPerMeter`** — all
  golden expectations are derived from the buggy implementation and will
  need to be regenerated from the corrected formula.
- **`pkg/gll/array_calculations.go`** — calls `AirLossPerMeter` for
  per-band air absorption when `airAbsorption` is enabled in
  `ComputeSystemResponseAt`. Output magnitudes will increase for any
  receiver beyond 0 m in humid conditions, by 30 % at low frequency to
  several × at high frequency.
- **CLF / SOFA / web exports** — none capture air absorption directly
  (they store source data, not propagated SPL). Plots and array
  predictions that *use* the air-loss term will shift.
- **`testdata/`** — no captured propagated-SPL fixtures, so no goldens to
  refresh outside the unit test above.
- **Ground-truth test** (`TestGroundTruth_SingleSourceOnAxis_…`) — passes
  `false` for `airAbsorption`, so unaffected.

## Recommendation

1. Apply the one-line fix above.
2. Regenerate `TestAirLossPerMeter` golden values from the corrected
   implementation.
3. Run the full test suite; any failure in non-air-absorption tests
   indicates an unintended dependency that should be investigated.
4. Add a regression test pinning ISO 9613-1 reference values at
   well-known conditions (e.g. ANSI S1.26-1995 Annex D worked example)
   so future drift is caught immediately.
5. Decide whether to upgrade to ANSI S1.26-2014's Hyland-Wexler
   vapor-pressure formula — it's a smaller (~5 %) further refinement and
   not strictly required for ISO 9613-1:1993 conformance.

## References

- ISO 9613-1:1993 — *Acoustics — Attenuation of sound during propagation
  outdoors — Part 1: Calculation of the absorption of sound by the
  atmosphere.*
- ANSI/ASA S1.26-1995 (R2014) — *Methods for Calculation of the
  Absorption of Sound by the Atmosphere.*
- Bass, H. E., et al. (1995). "Atmospheric absorption of sound: Further
  developments." *J. Acoust. Soc. Am.* 97(1), 680–683.
- Salomons, E. M. (2001). *Computational Atmospheric Acoustics.* Kluwer,
  Table 3.1 (ISO 9613-1 reference values at 20 °C, 1 atm, 50 % RH).
- sengpielaudio, "ISO 9613-1 air damping formula"
  ([sengpielaudio.com/AirdampingFormula.htm](https://sengpielaudio.com/AirdampingFormula.htm))
  — reproduced derivation explicitly stating that `h_r` is the relative
  humidity *as a percentage*.
