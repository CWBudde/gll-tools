package viz

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

func TestBuildFrequencyList(t *testing.T) {
	tests := []struct {
		name    string
		def     gll.LogSpectrumDefinition
		wantLen int
		wantNil bool
		checkFn func(t *testing.T, freqs []float64)
	}{
		{
			name:    "valid 3 bands per octave, 7 points from 100 Hz",
			def:     gll.LogSpectrumDefinition{BandsPerOctave: 3, StartFreq: 100, PointCount: 7},
			wantLen: 7,
			checkFn: func(t *testing.T, freqs []float64) {
				t.Helper()
				// First frequency should equal StartFreq
				if math.Abs(freqs[0]-100) > 0.001 {
					t.Errorf("first freq = %f, want 100", freqs[0])
				}
				// Frequencies should be strictly increasing
				for i := 1; i < len(freqs); i++ {
					if freqs[i] <= freqs[i-1] {
						t.Errorf("freqs[%d]=%f <= freqs[%d]=%f", i, freqs[i], i-1, freqs[i-1])
					}
				}
				// After 3 bands (one octave), frequency should double
				if math.Abs(freqs[3]-200) > 0.01 {
					t.Errorf("freqs[3] = %f, want ~200", freqs[3])
				}
			},
		},
		{
			name:    "single point",
			def:     gll.LogSpectrumDefinition{BandsPerOctave: 24, StartFreq: 1000, PointCount: 1},
			wantLen: 1,
			checkFn: func(t *testing.T, freqs []float64) {
				t.Helper()
				if math.Abs(freqs[0]-1000) > 0.001 {
					t.Errorf("single freq = %f, want 1000", freqs[0])
				}
			},
		},
		{
			name:    "zero point count",
			def:     gll.LogSpectrumDefinition{BandsPerOctave: 3, StartFreq: 100, PointCount: 0},
			wantNil: true,
		},
		{
			name:    "negative point count",
			def:     gll.LogSpectrumDefinition{BandsPerOctave: 3, StartFreq: 100, PointCount: -1},
			wantNil: true,
		},
		{
			name:    "zero bands per octave",
			def:     gll.LogSpectrumDefinition{BandsPerOctave: 0, StartFreq: 100, PointCount: 5},
			wantNil: true,
		},
		{
			name:    "zero start freq",
			def:     gll.LogSpectrumDefinition{BandsPerOctave: 3, StartFreq: 0, PointCount: 5},
			wantNil: true,
		},
		{
			name:    "negative start freq",
			def:     gll.LogSpectrumDefinition{BandsPerOctave: 3, StartFreq: -50, PointCount: 5},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFrequencyList(tt.def)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, got)
			}
		})
	}
}

func TestFindNearestFrequencyIndex(t *testing.T) {
	tests := []struct {
		name     string
		freqs    []float64
		targetHz float64
		want     int
	}{
		{
			name:     "exact match",
			freqs:    []float64{100, 200, 500, 1000, 2000},
			targetHz: 500,
			want:     2,
		},
		{
			name:     "between values closer to lower",
			freqs:    []float64{100, 200, 500, 1000, 2000},
			targetHz: 250,
			want:     1, // 200 is closer than 500
		},
		{
			name:     "between values closer to upper",
			freqs:    []float64{100, 200, 500, 1000, 2000},
			targetHz: 400,
			want:     2, // 500 is closer than 200
		},
		{
			name:     "empty slice",
			freqs:    []float64{},
			targetHz: 1000,
			want:     -1,
		},
		{
			name:     "single element exact",
			freqs:    []float64{1000},
			targetHz: 1000,
			want:     0,
		},
		{
			name:     "single element not exact",
			freqs:    []float64{1000},
			targetHz: 500,
			want:     0,
		},
		{
			name:     "target below range",
			freqs:    []float64{100, 200, 500},
			targetHz: 10,
			want:     0,
		},
		{
			name:     "target above range",
			freqs:    []float64{100, 200, 500},
			targetHz: 10000,
			want:     2,
		},
		{
			name:     "target zero or negative returns index 0",
			freqs:    []float64{100, 200, 500},
			targetHz: 0,
			want:     0,
		},
		{
			name:     "negative target returns index 0",
			freqs:    []float64{100, 200, 500},
			targetHz: -50,
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindNearestFrequencyIndex(tt.freqs, tt.targetHz)
			if got != tt.want {
				t.Errorf("FindNearestFrequencyIndex(%v, %f) = %d, want %d", tt.freqs, tt.targetHz, got, tt.want)
			}
		})
	}
}

func TestRenderPolarSVG(t *testing.T) {
	tests := []struct {
		name     string
		plot     PolarPlot
		wantErr  bool
		checkSVG func(t *testing.T, svg string)
	}{
		{
			name: "valid polar plot produces SVG",
			plot: PolarPlot{
				Width:       400,
				Height:      400,
				Title:       "Test Polar",
				FrequencyHz: 1000,
				AnglesDeg:   []float64{0, 90, 180, 270},
				Horizontal:  []float64{-10, -15, -20, -15},
				Vertical:    []float64{-10, -12, -25, -12},
			},
			checkSVG: func(t *testing.T, svg string) {
				t.Helper()
				if !strings.Contains(svg, "<svg") {
					t.Error("missing svg element")
				}
				if !strings.Contains(svg, "</svg>") {
					t.Error("missing closing svg tag")
				}
				if !strings.Contains(svg, "Test Polar") {
					t.Error("missing title")
				}
				if !strings.Contains(svg, "1.0 kHz") {
					t.Error("missing frequency label")
				}
			},
		},
		{
			name: "all NaN levels returns error",
			plot: PolarPlot{
				AnglesDeg:  []float64{0, 90},
				Horizontal: []float64{math.NaN(), math.NaN()},
				Vertical:   []float64{math.NaN(), math.NaN()},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := RenderPolarSVG(&buf, tt.plot)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.checkSVG != nil {
				tt.checkSVG(t, buf.String())
			}
		})
	}
}

func TestRenderResponseSVG(t *testing.T) {
	tests := []struct {
		name    string
		plot    ResponsePlot
		wantErr bool
	}{
		{
			name: "valid magnitude response",
			plot: ResponsePlot{
				Width:       600,
				Height:      400,
				Title:       "Test Response",
				Frequencies: []float64{100, 200, 500, 1000, 2000, 5000},
				Series:      []float64{-20, -18, -15, -10, -12, -16},
				Kind:        ResponseMagnitude,
			},
		},
		{
			name: "empty frequencies returns error",
			plot: ResponsePlot{
				Frequencies: []float64{},
				Series:      []float64{-10},
			},
			wantErr: true,
		},
		{
			name: "empty series returns error",
			plot: ResponsePlot{
				Frequencies: []float64{100},
				Series:      []float64{},
			},
			wantErr: true,
		},
		{
			name: "single frequency returns error (min==max)",
			plot: ResponsePlot{
				Frequencies: []float64{100},
				Series:      []float64{-10},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := RenderResponseSVG(&buf, tt.plot)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			svg := buf.String()
			if !strings.Contains(svg, "<svg") {
				t.Error("missing svg element")
			}
			if !strings.Contains(svg, "Test Response") {
				t.Error("missing title")
			}
		})
	}
}

func TestBuildResponseSeries(t *testing.T) {
	t.Run("nil response returns error", func(t *testing.T) {
		_, err := BuildResponseSeries(nil, nil, false)
		if err == nil {
			t.Error("expected error for nil response")
		}
	})

	t.Run("valid response builds series", func(t *testing.T) {
		resp := &gll.TransferFunction{
			Definition: gll.LogSpectrumDefinition{
				BandsPerOctave: 3,
				StartFreq:      100,
				PointCount:     7,
			},
			Level: []float64{-10, -12, -15, -10, -8, -12, -14},
			Phase: []float64{0, 0.5, 1.0, 1.5, 2.0, 2.5, 3.0},
		}
		series, err := BuildResponseSeries(nil, resp, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(series.Frequencies) != 7 {
			t.Errorf("frequencies len = %d, want 7", len(series.Frequencies))
		}
		if len(series.Level) != 7 {
			t.Errorf("level len = %d, want 7", len(series.Level))
		}
		if len(series.Phase) != 7 {
			t.Errorf("phase len = %d, want 7", len(series.Phase))
		}
		if len(series.PhaseWrapped) != 7 {
			t.Errorf("phase wrapped len = %d, want 7", len(series.PhaseWrapped))
		}
		if len(series.GroupDelayMs) != 7 {
			t.Errorf("group delay len = %d, want 7", len(series.GroupDelayMs))
		}
		if series.UsesOnAxis {
			t.Error("expected UsesOnAxis=false")
		}
	})
}

func TestResponseAtAngles(t *testing.T) {
	t.Run("nil balloon data returns nil", func(t *testing.T) {
		got := ResponseAtAngles(nil, 0, 0)
		if got != nil {
			t.Error("expected nil for nil balloon data")
		}
	})
}
