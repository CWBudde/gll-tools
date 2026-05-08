package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

// TestBuildFilterSpectrumForSource_NilGuards covers the six early-return
// branches at the top of the function.
func TestBuildFilterSpectrumForSource_NilGuards(t *testing.T) {
	t.Run("nil file", func(t *testing.T) {
		if got := buildFilterSpectrumForSource(nil, &gll.SourceDefinition{}, []string{"x"}); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
	t.Run("nil database", func(t *testing.T) {
		f := &gll.File{}
		if got := buildFilterSpectrumForSource(f, &gll.SourceDefinition{}, []string{"x"}); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
	t.Run("nil src def", func(t *testing.T) {
		f := &gll.File{Database: &gll.Database{}}
		if got := buildFilterSpectrumForSource(f, nil, []string{"x"}); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
	t.Run("empty groupKeys", func(t *testing.T) {
		f := &gll.File{Database: &gll.Database{}}
		if got := buildFilterSpectrumForSource(f, &gll.SourceDefinition{}, nil); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
	t.Run("nil balloon data", func(t *testing.T) {
		f := &gll.File{Database: &gll.Database{}}
		def := &gll.SourceDefinition{BalloonData: nil}
		if got := buildFilterSpectrumForSource(f, def, []string{"x"}); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
	t.Run("zero responses", func(t *testing.T) {
		f := &gll.File{Database: &gll.Database{}}
		def := &gll.SourceDefinition{BalloonData: &gll.BalloonData{}}
		if got := buildFilterSpectrumForSource(f, def, []string{"x"}); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}

// TestBuildFilterSpectrumForSource_LoopBranches drives the main loop:
//   - empty groupKey skipped
//   - duplicate key deduplicated
//   - unknown groupKey skipped (groupIndex < 0)
//
// The synthetic source has no matching filter-group keys, so the loop
// completes without producing a combined spectrum (returns nil). Coverage
// of the loop entry, the empty/seen/lookup branches is the value here.
func TestBuildFilterSpectrumForSource_LoopBranches(t *testing.T) {
	def := gll.LogSpectrumDefinition{BandsPerOctave: 1, StartFreq: 1000, PointCount: 1}
	tf := gll.TransferFunction{
		Definition: def,
		Level:      []float64{0},
		Phase:      []float64{0},
	}
	srcDef := &gll.SourceDefinition{
		BalloonData: &gll.BalloonData{
			AngularResolution: gll.ResolutionDescriptor{},
			Responses:         []gll.TransferFunction{tf},
		},
	}
	file := &gll.File{Database: &gll.Database{}}

	// All branches: empty key, dup, unknown key.
	got := buildFilterSpectrumForSource(file, srcDef, []string{"", "groupA", "groupA", "unknown"})
	if got != nil {
		t.Errorf("got non-nil result %v, want nil (no filter groups in synthetic file)", got)
	}
}

// TestBuildFilterSpectrumForSource_RealFixture loads SphereLine19 (the only
// fixture in the repo with populated filter groups), eagerly loads balloon
// responses, and drives buildFilterSpectrumForSource against each
// (source-def, filter-group) pair. The function may or may not return a
// combined spectrum depending on whether the balloon's frequency grid
// matches the filter's; either outcome exercises the inner-loop body.
func TestBuildFilterSpectrumForSource_RealFixture(t *testing.T) {
	path := filepath.FromSlash("../../legacy/viewer/SphereLine19_AsyFed.gll")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}

	r := bytes.NewReader(data)
	file, err := gll.Parse(r)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if file.Database == nil || len(file.Database.SourceDefinitions) == 0 {
		t.Fatal("fixture has no source definitions")
	}

	for _, src := range file.Database.SourceDefinitions {
		if src.Definition == nil || src.Definition.BalloonData == nil {
			continue
		}
		// Lazy-load balloon responses (Parse leaves Responses empty).
		if err := gll.LoadBalloonResponses(r, src.Definition.BalloonData); err != nil {
			t.Fatalf("LoadBalloonResponses(%s): %v", src.Key, err)
		}
		var keys []string
		for _, g := range file.Database.FilterGroups {
			keys = append(keys, g.Key)
		}
		// Drive every group key against every source. We only assert no
		// panic; the function's mid-flight branches all execute.
		_ = buildFilterSpectrumForSource(file, src.Definition, keys)
	}
}

// TestComputeArrayResponseData_BadConfigJSON drives the JSON-unmarshal error
// branch that the existing parse-error test doesn't reach (it sends a bad
// GLL with empty config, so parse fails first and config is never read).
func TestComputeArrayResponseData_BadConfigJSON(t *testing.T) {
	// A minimal valid-ish GLL is hard to synthesize inline, so use the
	// SphereLine19 fixture (it parses cleanly).
	path := filepath.FromSlash("../../legacy/viewer/SphereLine19_AsyFed.gll")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	res := computeArrayResponseData(data, "not-json")
	if res.Success {
		t.Fatal("expected failure on bad JSON")
	}
	if res.Error == "" {
		t.Fatal("expected error message")
	}
}

// TestComputeArrayBalloonData_BadConfigJSON same idea for the balloon path.
func TestComputeArrayBalloonData_BadConfigJSON(t *testing.T) {
	path := filepath.FromSlash("../../legacy/viewer/SphereLine19_AsyFed.gll")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	res := computeArrayBalloonData(data, "not-json", nil)
	if res.Success {
		t.Fatal("expected failure on bad JSON")
	}
	if res.Error == "" {
		t.Fatal("expected error message")
	}
}

// TestComputeArrayResponseForFile_NoElements drives the "no valid elements"
// branch in computeArrayResponseForFile.
func TestComputeArrayResponseForFile_NoElements(t *testing.T) {
	file := &gll.File{Database: &gll.Database{}}
	req := ArrayResponseRequest{Elements: nil}
	res := computeArrayResponseForFile(file, nil, req)
	if res.Success {
		t.Fatal("expected failure when no elements")
	}
}
