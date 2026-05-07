package cmd

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cwbudde/gll-tools/internal/viz"
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
