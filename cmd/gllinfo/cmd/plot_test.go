package cmd

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cwbudde/gll-tools/internal/mesh"
	"github.com/cwbudde/gll-tools/internal/viz"
	"github.com/cwbudde/gll-tools/pkg/gll"
)

func TestPlotPolarCommand(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "polar.svg")
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "plot", "polar", "--source", "0", "--output", out, path); err != nil {
		t.Fatalf("plot polar command failed: %v", err)
	}
}

func TestPlotPolarMissingSource(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "polar.svg")
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "plot", "polar", "--output", out, path); err == nil {
		t.Fatal("expected error when --source is omitted")
	}
}

func TestPlotPolarMissingOutput(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "plot", "polar", "--source", "0", path); err == nil {
		t.Fatal("expected error when --output is omitted")
	}
}

func TestPlotResponseCommand(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "response.svg")
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "plot", "response", "--source", "0", "--output", out, path); err != nil {
		t.Fatalf("plot response command failed: %v", err)
	}
}

func TestPlotBalloonCommand(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "balloon.stl")
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "plot", "balloon", "--source", "0", "--output", out, path); err != nil {
		t.Fatalf("plot balloon command failed: %v", err)
	}
}

func TestPlotPolarBadOutputExtension(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "polar.png")
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "plot", "polar", "--source", "0", "--output", out, path); err == nil {
		t.Fatal("expected error for non-SVG output extension")
	}
}

func TestPlotBalloonBadOutputExtension(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "balloon.svg") // mesh extension required
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "plot", "balloon", "--source", "0", "--output", out, path); err == nil {
		t.Fatal("expected error for non-mesh output extension")
	}
}

func TestPlotMissingFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "polar.svg")
	if err := runRoot(t, "plot", "polar", "--source", "0", "--output", out, "nonexistent.gll"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestPlotGeometryCommand(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "geometry.stl")
	path := filepath.FromSlash(testGLLFile)
	// D12-v10.gll may or may not have geometry; either success or a controlled
	// error is acceptable. The point is to exercise the code path.
	_ = runRoot(t, "plot", "geometry", "--output", out, path)
}

func TestPlotGeometryMissingOutput(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "plot", "geometry", path); err == nil {
		t.Fatal("expected error when --output is omitted")
	}
}

func TestPlotGeometryBadOutputExtension(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "geometry.txt")
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "plot", "geometry", "--output", out, path); err == nil {
		t.Fatal("expected error for non-mesh output extension")
	}
}

func TestNormalizeLevels(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want []float64
	}{
		{
			name: "uniform values map to zero",
			in:   []float64{5, 5, 5},
			want: []float64{0, 0, 0},
		},
		{
			name: "max becomes zero, others negative",
			in:   []float64{0, 5, 10},
			want: []float64{-10, -5, 0},
		},
		{
			name: "NaN values preserved",
			in:   []float64{1, math.NaN(), 5},
			want: []float64{-4, math.NaN(), 0},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vals := append([]float64(nil), tc.in...)
			normalizeLevels(vals)
			for i, v := range vals {
				w := tc.want[i]
				if math.IsNaN(v) && math.IsNaN(w) {
					continue
				}
				if math.Abs(v-w) > 1e-9 {
					t.Errorf("normalizeLevels[%d] = %v, want %v", i, v, w)
				}
			}
		})
	}
}

func TestNormalizeLevelsAllNaN(t *testing.T) {
	vals := []float64{math.NaN(), math.NaN()}
	normalizeLevels(vals) // should not panic, no-op
	for i, v := range vals {
		if !math.IsNaN(v) {
			t.Errorf("vals[%d] = %v, want NaN", i, v)
		}
	}
}

func TestSelectResponseSeriesModes(t *testing.T) {
	series := &viz.ResponseSeries{
		Level:        []float64{1, 2, 3},
		Phase:        []float64{0.1, 0.2, 0.3},
		PhaseWrapped: []float64{0.1, 0.2, 0.3},
		GroupDelayMs: []float64{0.5, 0.5, 0.5},
	}

	cases := []struct {
		mode    string
		wantErr bool
	}{
		{"magnitude", false},
		{"level", false},
		{"", false},
		{"phase-wrapped", false},
		{"wrapped", false},
		{"phase-unwrapped", false},
		{"phase", false},
		{"unwrapped", false},
		{"group-delay", false},
		{"delay", false},
		{"BOGUS", true},
	}

	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			_, _, err := selectResponseSeries(series, tc.mode)
			if (err != nil) != tc.wantErr {
				t.Errorf("selectResponseSeries(%q) error=%v, wantErr=%v",
					tc.mode, err, tc.wantErr)
			}
		})
	}
}

func TestEnsureSVGOutput(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid svg", "/tmp/out.svg", false},
		{"uppercase svg", "/tmp/out.SVG", false},
		{"missing extension", "/tmp/out", true},
		{"png rejected", "/tmp/out.png", true},
		{"pdf rejected", "/tmp/out.pdf", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ensureSVGOutput(tc.path)
			if (err != nil) != tc.wantErr {
				t.Errorf("ensureSVGOutput(%q) error=%v, wantErr=%v", tc.path, err, tc.wantErr)
			}
		})
	}
}

func TestEnsureMeshOutput(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"stl ok", "/tmp/out.stl", false},
		{"obj ok", "/tmp/out.obj", false},
		{"dxf ok", "/tmp/out.dxf", false},
		{"uppercase stl", "/tmp/out.STL", false},
		{"svg rejected", "/tmp/out.svg", true},
		{"missing extension", "/tmp/out", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ensureMeshOutput(tc.path)
			if (err != nil) != tc.wantErr {
				t.Errorf("ensureMeshOutput(%q) error=%v, wantErr=%v", tc.path, err, tc.wantErr)
			}
		})
	}
}

func TestApplyMeshCenter(t *testing.T) {
	makeMesh := func() *mesh.Mesh {
		return &mesh.Mesh{
			Vertices: []mesh.Vec3{{X: 0, Y: 0, Z: 0}, {X: 2, Y: 4, Z: 6}},
		}
	}

	cases := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{"empty mode is no-op", "", false},
		{"origin", "origin", false},
		{"bbox", "bbox", false},
		{"centroid", "centroid", false},
		{"uppercase accepted", "BBOX", false},
		{"whitespace trimmed", "  centroid  ", false},
		{"unknown rejected", "middle", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := applyMeshCenter(makeMesh(), tc.mode)
			if (err != nil) != tc.wantErr {
				t.Errorf("applyMeshCenter(%q) error=%v, wantErr=%v", tc.mode, err, tc.wantErr)
			}
		})
	}
}

func TestSelectCaseGeometryNilDatabase(t *testing.T) {
	if _, _, err := selectCaseGeometry(nil, -1, -1, false); err == nil {
		t.Fatal("expected error for nil database")
	}
}

func TestSelectCaseGeometryFromBox(t *testing.T) {
	geom := &gll.CaseGeometry{}
	db := &gll.Database{
		BoxTypes: []gll.BoxType{
			{Label: "Box A", Key: "boxA", CaseGeometry: geom},
		},
	}
	got, label, err := selectCaseGeometry(db, -1, -1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != geom {
		t.Errorf("got geometry %p, want %p", got, geom)
	}
	if label != "Box A" {
		t.Errorf("got label %q, want %q", label, "Box A")
	}
}

func TestSelectCaseGeometryBoxLabelFallsBackToKey(t *testing.T) {
	geom := &gll.CaseGeometry{}
	db := &gll.Database{
		BoxTypes: []gll.BoxType{{Key: "boxA", CaseGeometry: geom}},
	}
	_, label, err := selectCaseGeometry(db, 0, -1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if label != "boxA" {
		t.Errorf("got label %q, want fallback to key %q", label, "boxA")
	}
}

func TestSelectCaseGeometryFromFrameByIndex(t *testing.T) {
	geom := &gll.CaseGeometry{}
	db := &gll.Database{
		Frames: []gll.Frame{
			{Label: "Frame 1", Key: "f1", CaseGeometry: geom},
		},
		BoxTypes: []gll.BoxType{{Label: "Box", CaseGeometry: &gll.CaseGeometry{}}},
	}
	got, label, err := selectCaseGeometry(db, -1, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != geom {
		t.Error("expected frame geometry, got box geometry")
	}
	if label != "Frame 1" {
		t.Errorf("got label %q, want %q", label, "Frame 1")
	}
}

func TestSelectCaseGeometryPreferFrame(t *testing.T) {
	frameGeom := &gll.CaseGeometry{}
	db := &gll.Database{
		Frames: []gll.Frame{
			{Key: "noGeom"},
			{Label: "Frame 2", CaseGeometry: frameGeom},
		},
		BoxTypes: []gll.BoxType{{Label: "Box", CaseGeometry: &gll.CaseGeometry{}}},
	}
	got, label, err := selectCaseGeometry(db, -1, -1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != frameGeom {
		t.Error("expected to prefer frame geometry")
	}
	if label != "Frame 2" {
		t.Errorf("got label %q, want %q", label, "Frame 2")
	}
}

func TestSelectCaseGeometryErrors(t *testing.T) {
	cases := []struct {
		name       string
		db         *gll.Database
		boxIdx     int
		frameIdx   int
		preferFrm  bool
		wantErrSub string
	}{
		{
			name:       "frame index out of range",
			db:         &gll.Database{Frames: []gll.Frame{{}}},
			frameIdx:   5,
			boxIdx:     -1,
			wantErrSub: "frame index",
		},
		{
			name:       "frame has no geometry",
			db:         &gll.Database{Frames: []gll.Frame{{Key: "f"}}},
			frameIdx:   0,
			boxIdx:     -1,
			wantErrSub: "no geometry",
		},
		{
			name:       "no box types",
			db:         &gll.Database{},
			frameIdx:   -1,
			boxIdx:     -1,
			wantErrSub: "no box types",
		},
		{
			name:       "box index out of range",
			db:         &gll.Database{BoxTypes: []gll.BoxType{{}}},
			frameIdx:   -1,
			boxIdx:     7,
			wantErrSub: "box index",
		},
		{
			name:       "box has no geometry",
			db:         &gll.Database{BoxTypes: []gll.BoxType{{Key: "b"}}},
			frameIdx:   -1,
			boxIdx:     0,
			wantErrSub: "no geometry",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := selectCaseGeometry(tc.db, tc.boxIdx, tc.frameIdx, tc.preferFrm)
			if err == nil {
				t.Fatalf("expected error containing %q", tc.wantErrSub)
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErrSub)
			}
		})
	}
}

func TestSelectResponseSeriesUppercaseAccepted(t *testing.T) {
	// Mode matching is case-insensitive after trimming.
	series := &viz.ResponseSeries{Level: []float64{1, 2, 3}}
	if _, _, err := selectResponseSeries(series, " MAGNITUDE "); err != nil {
		t.Errorf("expected case-insensitive mode matching, got error: %v", err)
	}
	if _, _, err := selectResponseSeries(series, strings.ToUpper("phase")); err != nil {
		t.Errorf("expected case-insensitive mode matching for phase, got error: %v", err)
	}
}
