package gll

// Ground-truth tests for ComputeSystemResponseAt against real GLL fixtures.
//
// Goal: verify that "place a single source at the origin, listener at 1 m on-axis,
// no air absorption, no filters" produces an SPL that matches the source's own
// declared on-axis spectrum (at the GLL's measurement distance, typically 1 m).
//
// The investigation that produced this test is summarized below. It also resolves
// the open contract from docs/acoustic-model.md ("Source Response Components").
//
// CONCLUSION FROM THE FIXTURES
//
//   SourceDefinition.OnAxisLevel == 94.00 dB universally — this is the IEC
//   reference (94 dB SPL = 1 Pa, 0 dBFS for an SPL meter).
//
//   Two encoding conventions appear in the wild:
//
//     CONVENTION A — relative directivity + separate spectrum (most fixtures)
//       BalloonData.Responses store relative directivity in dB, on-axis = 0 dB.
//       SourceDefinition.OnAxisSpectrum holds the absolute on-axis SPL in dB
//       at the measurement distance (typically 1 m). Peak realistic values
//       range from ~85 to ~105 dB SPL.
//       Examples: D12, D20, APS, HOPS7-Pro, all Coda Audio + Fohhn fixtures.
//
//     CONVENTION B — absolute SPL baked into the balloon (a few examples)
//       BalloonData.Responses store absolute SPL directly. OnAxisSpectrum is
//       absent or zero.
//       Examples: example-cl.gll (balloon front pole midband ≈ 90 dB).
//
//   The 3Way-LR.gll fixture has a zero OnAxisSpectrum and (likely) zero
//   balloon — it appears to be a placeholder / template with no real data.
//
// DERIVED INVARIANT
//
//   At a receiver placed 1 m in front of a single source, with no filters,
//   no element gain, and no air absorption:
//
//       SPL(receiver, f) ≈ OnAxisSpectrum.Level[f] + 20·log10(1)            (1)
//                       ≈ OnAxisSpectrum.Level[f]                            (2)
//
//   The "+ 20·log10(1)" term is the 1/r spreading at d=1 m and is exactly 0.
//
// CURRENT IMPLEMENTATION
//
//   computeElementResponseAt does:
//
//       response  = balloon @ (meridian, parallel)             // relative dB
//       response *= filterSpectrum                              // skipped here
//       response += elem.Gain                                   // 0 here
//       response *= responseAtGLLAngles(0, 0)                   // BUG (see below)
//       response += 20·log10(1/d)                               // 0 at d=1m
//       (no application of OnAxisSpectrum anywhere)
//
//   The `Multiply(onAxis)` step is wrong for both conventions:
//
//     • Convention A: balloon front pole = 0 dB → multiplication is a no-op.
//       OnAxisSpectrum is never applied. Result at 1 m on-axis ≈ 0 dB SPL,
//       missing ~95 dB of source sensitivity.
//     • Convention B: balloon front pole = absolute SPL (~90 dB) → multiplication
//       doubles it to ~180 dB. OnAxisSpectrum is empty so the missing-spectrum
//       half of the bug doesn't show, but the on-axis is now ~90 dB too loud.
//
//   docs/acoustic-model.md → "Source Response Components" listed two open
//   questions:
//     • Whether BalloonData.Responses are relative or absolute → BOTH appear
//       in the wild; the parser cannot tell from the file alone. A heuristic
//       (e.g. "balloon front pole < 30 dB AND OnAxisSpectrum non-empty
//       indicates Convention A") is needed.
//     • Whether array response should use SourceDefinition.OnAxisSpectrum
//       for on-axis normalization → YES (when present); the current code
//       never uses OnAxisSpectrum.
//
// PROPOSED FIX
//
//   Replace `response.Multiply(onAxis)` with:
//
//       if srcDef.OnAxisSpectrum != nil && len(srcDef.OnAxisSpectrum.Level) > 0 {
//           // Convention A. Frequency-grid alignment may be required.
//           response.Multiply(srcDef.OnAxisSpectrum)
//       }
//       // Convention B: do nothing — the balloon already encodes absolute SPL.
//
// STATUS: RESOLVED
//
//   The proposed fix has landed: computeElementResponseAt now applies
//   SourceDefinition.OnAxisSpectrum when present and grid-aligned (see
//   applyOnAxisSpectrum in array_calculations.go). This file's tests are now
//   regression gates rather than bug reports.
//
//   TestGroundTruth_SingleSourceOnAxis_MatchesOnAxisSpectrum asserts that
//   D12 at 1 m on-axis returns ~96.23 dB SPL (its OnAxisSpectrum midband).
//
//   TestGroundTruth_OnAxisSpectrum_ConventionAcrossFixtures inventories which
//   fixtures fall into Convention A vs Convention B, logging counts. It only
//   fails if the dominant convention disappears from the fixture set.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestGroundTruth_SingleSourceOnAxis_MatchesOnAxisSpectrum verifies that a
// single GLL source, computed by the array engine at 1 m on-axis with no
// filters and no air loss, produces SPL equal to its OnAxisSpectrum.
//
// This is a regression gate for the on-axis SPL fix.
func TestGroundTruth_SingleSourceOnAxis_MatchesOnAxisSpectrum(t *testing.T) {
	// D12 is the cleanest fixture: single source, single coaxial driver,
	// well-defined OnAxisSpectrum from the parser. Other 1-source files
	// (Hybrid-1, IG-80, IG-100, LX-10, LX-20, LX-60) work the same way.
	const fixture = "D12-v10.gll"

	f, err := os.Open(filepath.Join("..", "..", "testdata", "gll", fixture))
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	t.Cleanup(func() { f.Close() })

	gllFile, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if gllFile.Database == nil || len(gllFile.Database.SourceDefinitions) == 0 {
		t.Skip("no source definitions")
	}

	src := gllFile.Database.SourceDefinitions[0].Definition
	if src == nil {
		t.Fatal("nil source definition")
	}
	if src.BalloonData == nil {
		t.Fatal("nil balloon data")
	}
	if err := LoadBalloonResponses(f, src.BalloonData); err != nil {
		t.Fatalf("LoadBalloonResponses: %v", err)
	}
	if src.OnAxisSpectrum == nil || len(src.OnAxisSpectrum.Level) == 0 {
		t.Skipf("%s has no OnAxisSpectrum to compare against", fixture)
	}

	// Pick a midband bin for a clean comparison. The OnAxisSpectrum should
	// have a realistic SPL at this bin (~95 dB for D12 around 1 kHz).
	midBand := len(src.OnAxisSpectrum.Level) / 2
	expectedSPL := src.OnAxisSpectrum.Level[midBand]
	if expectedSPL < 60 {
		t.Skipf("OnAxisSpectrum midband level %.2f dB is too low to be a "+
			"useful reference (expected something like 80-100 dB SPL)", expectedSPL)
	}

	// Build a single-element array at the origin.
	cfg := &ArrayConfig{
		Elements: []ArrayElement{{
			Position:   Vector3D{},
			SourceDefs: []*SourceDefinition{src},
		}},
	}

	// Receiver at 1 m on-axis (+X), no filters, no air absorption.
	resp := ComputeSystemResponseAt(cfg, Vector3D{X: 1}, AirProperties{Speed: 343}, false)
	if resp == nil {
		t.Fatal("ComputeSystemResponseAt returned nil")
	}
	if len(resp.Level) <= midBand {
		t.Fatalf("response has %d bands, need at least %d", len(resp.Level), midBand+1)
	}

	gotSPL := resp.Level[midBand]
	const tolerance = 1.0 // dB; balloon-front interpolation may not be exactly 0
	delta := gotSPL - expectedSPL

	t.Logf("fixture          : %s", fixture)
	t.Logf("source           : %s", src.Label)
	t.Logf("expected SPL @1m : %.2f dB  (from OnAxisSpectrum.Level[%d])", expectedSPL, midBand)
	t.Logf("computed SPL @1m : %.2f dB  (from ComputeSystemResponseAt)", gotSPL)
	t.Logf("Δ                : %+.2f dB (tolerance ±%.1f dB)", delta, tolerance)

	if math.Abs(delta) > tolerance {
		t.Errorf(
			"SPL mismatch: got %.2f dB, want %.2f ± %.1f dB.\n"+
				"This is the docs/acoustic-model.md open contract: the array engine "+
				"does not apply SourceDefinition.OnAxisSpectrum, so it returns the "+
				"relative directivity (≈0 dB on-axis) instead of absolute SPL. "+
				"Fix: in computeElementResponseAt, after directivity interpolation, "+
				"multiply by srcDef.OnAxisSpectrum (frequency-grid alignment "+
				"required) instead of the balloon's front-pole spectrum.",
			gotSPL, expectedSPL, tolerance,
		)
	}
}

// TestGroundTruth_OnAxisSpectrum_ConventionAcrossFixtures inventories the
// fixture set into Convention A (relative balloon + OnAxisSpectrum) versus
// Convention B (absolute balloon, empty OnAxisSpectrum) and a placeholder
// bucket (both zero). The test only fails if Convention A is missing from
// the fixture set entirely — that would invalidate the proposed fix design.
func TestGroundTruth_OnAxisSpectrum_ConventionAcrossFixtures(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "gll")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("testdata dir not available: %v", err)
	}

	var convA, convB, placeholder, unknown int
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".gll" {
			continue
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}

		gllFile, err := Parse(f)
		if err != nil {
			f.Close()
			continue
		}
		if gllFile.Database == nil {
			f.Close()
			continue
		}

		for _, item := range gllFile.Database.SourceDefinitions {
			src := item.Definition
			if src == nil || src.BalloonData == nil {
				continue
			}
			if err := LoadBalloonResponses(f, src.BalloonData); err != nil {
				continue
			}
			if len(src.BalloonData.Responses) == 0 ||
				len(src.BalloonData.Responses[0].Level) == 0 {
				continue
			}

			front := &src.BalloonData.Responses[0]
			fmid := front.Level[len(front.Level)/2]
			balloonRelative := math.Abs(fmid) < 5.0
			balloonAbsolute := fmid > 50

			hasSpectrum := src.OnAxisSpectrum != nil && len(src.OnAxisSpectrum.Level) > 0
			spectrumRealistic := false
			if hasSpectrum {
				v := src.OnAxisSpectrum.Level[len(src.OnAxisSpectrum.Level)/2]
				spectrumRealistic = v >= 50 && v <= 130
			}

			switch {
			case balloonRelative && spectrumRealistic:
				convA++
			case balloonAbsolute && !spectrumRealistic:
				convB++
			case balloonRelative && !spectrumRealistic:
				placeholder++
			default:
				unknown++
				t.Logf("UNKNOWN convention: %s src=%q balloonFrontMid=%.2f dB "+
					"OnAxisSpectrumMid=%v", e.Name(), src.Label, fmid,
					func() string {
						if !hasSpectrum {
							return "<absent>"
						}
						return fmt.Sprintf("%.2f dB",
							src.OnAxisSpectrum.Level[len(src.OnAxisSpectrum.Level)/2])
					}())
			}
		}
		f.Close()
	}

	t.Logf("Convention inventory across fixtures:")
	t.Logf("  A  (relative balloon + realistic OnAxisSpectrum) : %d sources", convA)
	t.Logf("  B  (absolute balloon, no/zero OnAxisSpectrum)    : %d sources", convB)
	t.Logf("  P  (placeholder: zero balloon + zero spectrum)   : %d sources", placeholder)
	t.Logf("  ?  (unknown / hybrid)                            : %d sources", unknown)

	if convA == 0 {
		t.Errorf("no Convention A sources found in fixtures; the proposed " +
			"OnAxisSpectrum fix has no test coverage")
	}
}
