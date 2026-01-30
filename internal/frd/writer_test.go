package frd

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestWriteResponse(t *testing.T) {
	tests := []struct {
		name        string
		frequencies []float64
		levels      []float64
		phases      []float64
		wantErr     bool
		wantLines   []string
	}{
		{
			name:        "basic response",
			frequencies: []float64{100.0, 200.0, 400.0},
			levels:      []float64{-6.0, -3.0, 0.0},
			phases:      []float64{0.0, math.Pi / 2, math.Pi},
			wantErr:     false,
			wantLines: []string{
				"100.000  -6.00  0.00",
				"200.000  -3.00  90.00",
				"400.000  0.00  180.00",
			},
		},
		{
			name:        "single point",
			frequencies: []float64{1000.0},
			levels:      []float64{-12.5},
			phases:      []float64{-math.Pi / 4},
			wantErr:     false,
			wantLines: []string{
				"1000.000  -12.50  -45.00",
			},
		},
		{
			name:        "fractional frequencies",
			frequencies: []float64{125.5, 250.25},
			levels:      []float64{1.23, -4.56},
			phases:      []float64{0.1, -0.2},
			wantErr:     false,
			wantLines: []string{
				"125.500  1.23  5.73",
				"250.250  -4.56  -11.46",
			},
		},
		{
			name:        "empty data",
			frequencies: []float64{},
			levels:      []float64{},
			phases:      []float64{},
			wantErr:     true,
		},
		{
			name:        "mismatched level length",
			frequencies: []float64{100.0, 200.0},
			levels:      []float64{-6.0},
			phases:      []float64{0.0, 0.0},
			wantErr:     true,
		},
		{
			name:        "mismatched phase length",
			frequencies: []float64{100.0, 200.0},
			levels:      []float64{-6.0, -3.0},
			phases:      []float64{0.0},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteResponse(&buf, tt.frequencies, tt.levels, tt.phases)

			if (err != nil) != tt.wantErr {
				t.Errorf("WriteResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got := strings.TrimSpace(buf.String())
			want := strings.Join(tt.wantLines, "\n")

			if got != want {
				t.Errorf("WriteResponse() output mismatch:\ngot:\n%s\n\nwant:\n%s", got, want)
			}
		})
	}
}

func TestWriteResponse_PhaseConversion(t *testing.T) {
	// Verify phase is correctly converted from radians to degrees
	var buf bytes.Buffer
	frequencies := []float64{1000.0}
	levels := []float64{0.0}
	phases := []float64{2 * math.Pi} // Full circle in radians

	err := WriteResponse(&buf, frequencies, levels, phases)
	if err != nil {
		t.Fatalf("WriteResponse() failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	expected := "1000.000  0.00  360.00"

	if output != expected {
		t.Errorf("Phase conversion incorrect:\ngot:  %s\nwant: %s", output, expected)
	}
}
