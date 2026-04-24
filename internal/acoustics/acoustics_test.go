package acoustics

import (
	"math"
	"testing"
)

// TestDefaultAirProperties tests standard air conditions
func TestDefaultAirProperties(t *testing.T) {
	temp, humidity, speed := DefaultAirProperties()

	if temp != 20.0 {
		t.Errorf("Expected temperature 20.0, got %f", temp)
	}
	if humidity != 0.5 {
		t.Errorf("Expected humidity 0.5, got %f", humidity)
	}
	if speed != 343.0 {
		t.Errorf("Expected speed 343.0, got %f", speed)
	}
}

// TestAirLossPerMeter tests air absorption calculations
func TestAirLossPerMeter(t *testing.T) {
	tests := []struct {
		name        string
		freq        float64
		temperature float64
		humidity    float64
		pressure    float64
		wantMin     float64
		wantMax     float64
	}{
		{"1kHz 20C 50% humidity", 1000, 20, 0.5, ReferencePressureKPa, 0.0034, 0.0036},
		{"10kHz 20C 50% humidity", 10000, 20, 0.5, ReferencePressureKPa, 0.019, 0.020},
		{"1kHz dry air", 1000, 20, 0.0, ReferencePressureKPa, 0.0015, 0.0017},
		{"1kHz humid air", 1000, 20, 1.0, ReferencePressureKPa, 0.0049, 0.0052},
		{"10kHz cold dry air", 10000, 0, 0.2, ReferencePressureKPa, 0.008, 0.010},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loss := AirLossPerMeter(tt.freq, tt.temperature, tt.humidity, tt.pressure)
			if loss < tt.wantMin || loss > tt.wantMax {
				t.Errorf("AirLossPerMeter(%f, %f, %f, %f) = %f, want between %f and %f",
					tt.freq, tt.temperature, tt.humidity, tt.pressure, loss, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestMapMeridianBySymmetry tests symmetry mapping
func TestMapMeridianBySymmetry(t *testing.T) {
	tests := []struct {
		name     string
		angle    float64
		symmetry int
		want     float64
	}{
		// SymmetryNone (0)
		{"None: 45°", 45, 0, 45},
		{"None: 180°", 180, 0, 180},
		{"None: 270°", 270, 0, 270},

		// SymmetryVertical (1) - 2-fold vertical
		{"Vertical: front", 90, 1, 90},
		{"Vertical: back mirror", 270, 1, 90},

		// SymmetryHorizontal (2) - 2-fold horizontal
		{"Horizontal: right", 90, 2, 0},
		{"Horizontal: front", 0, 2, 90},
		{"Horizontal: left", 270, 2, 180}, // 270-90=180, which is >=180, so 360-180=180

		// SymmetryQuarter (3) - 4-fold
		{"Quarter: 0-90°", 45, 3, 45},
		{"Quarter: 90-180° mirror", 135, 3, 45},
		{"Quarter: 180-270° offset", 225, 3, 45},
		{"Quarter: 270-360° mirror", 315, 3, 45},

		// SymmetryAxial (4)
		{"Axial: always 0°", 90, 4, 0},
		{"Axial: always 0°", 180, 4, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapMeridianBySymmetry(tt.angle, tt.symmetry)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("MapMeridianBySymmetry(%f, %d) = %f, want %f",
					tt.angle, tt.symmetry, got, tt.want)
			}
		})
	}
}

// TestMeridianCount tests meridian sample count calculation
func TestMeridianCount(t *testing.T) {
	tests := []struct {
		name     string
		step     float64
		symmetry int
		want     int
	}{
		{"None 5° step", 5, 0, 72},
		{"Vertical 5° step", 5, 1, 37},
		{"Horizontal 5° step", 5, 2, 37},
		{"Quarter 5° step", 5, 3, 19},
		{"Axial any step", 5, 4, 1},
		{"Zero step", 0, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MeridianCount(tt.step, tt.symmetry)
			if got != tt.want {
				t.Errorf("MeridianCount(%f, %d) = %d, want %d",
					tt.step, tt.symmetry, got, tt.want)
			}
		})
	}
}

// TestParallelCount tests parallel sample count calculation
func TestParallelCount(t *testing.T) {
	tests := []struct {
		name          string
		step          float64
		frontHalfOnly bool
		want          int
	}{
		{"Full sphere 5° step", 5, false, 37},
		{"Front half 5° step", 5, true, 19},
		{"Full sphere 10° step", 10, false, 19},
		{"Zero step", 0, false, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParallelCount(tt.step, tt.frontHalfOnly)
			if got != tt.want {
				t.Errorf("ParallelCount(%f, %t) = %d, want %d",
					tt.step, tt.frontHalfOnly, got, tt.want)
			}
		})
	}
}

// TestResponseIndex tests balloon grid indexing
func TestResponseIndex(t *testing.T) {
	parCount := 37 // 5° step, full sphere

	tests := []struct {
		name          string
		merIdx        int
		parIdx        int
		parCount      int
		frontHalfOnly bool
		want          int
	}{
		// Poles are stored at indices 0 and last
		{"Front pole mer=0", 0, 0, parCount, false, 0},
		{"Front pole mer=5", 5, 0, parCount, false, 0}, // Always index 0
		{"Back pole mer=0", 0, 36, parCount, false, 36},
		{"Back pole mer=5", 5, 36, parCount, false, 36}, // Always index 36

		// First meridian (mer=0) stores all parallels
		{"Mer 0 par 1", 0, 1, parCount, false, 1},
		{"Mer 0 par 18", 0, 18, parCount, false, 18},

		// Subsequent meridians skip poles
		{"Mer 1 par 1", 1, 1, parCount, false, 37}, // 37 + (1-1)*35 + (1-1)
		{"Mer 1 par 2", 1, 2, parCount, false, 38}, // 37 + (1-1)*35 + (2-1)
		{"Mer 2 par 1", 2, 1, parCount, false, 72}, // 37 + (2-1)*35 + (1-1)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResponseIndex(tt.merIdx, tt.parIdx, tt.parCount, tt.frontHalfOnly)
			if got != tt.want {
				t.Errorf("ResponseIndex(%d, %d, %d, %t) = %d, want %d",
					tt.merIdx, tt.parIdx, tt.parCount, tt.frontHalfOnly, got, tt.want)
			}
		})
	}
}

// TestClampParallelIndex tests index clamping
func TestClampParallelIndex(t *testing.T) {
	parCount := 37

	tests := []struct {
		idx  int
		want int
	}{
		{-5, 0},
		{0, 0},
		{18, 18},
		{36, 36},
		{40, 36},
	}

	for _, tt := range tests {
		got := ClampParallelIndex(tt.idx, parCount)
		if got != tt.want {
			t.Errorf("ClampParallelIndex(%d, %d) = %d, want %d",
				tt.idx, parCount, got, tt.want)
		}
	}
}

// TestBilinearWeights tests bilinear interpolation weight calculation
func TestBilinearWeights(t *testing.T) {
	tests := []struct {
		name   string
		merIdx float64
		parIdx float64
	}{
		{"Integer indices", 2.0, 3.0},
		{"Half fractional", 2.5, 3.5},
		{"Quarter fractional", 2.25, 3.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w00, w01, w10, w11 := BilinearWeights(tt.merIdx, tt.parIdx)

			// Weights must sum to 1.0
			sum := w00 + w01 + w10 + w11
			if math.Abs(sum-1.0) > 1e-10 {
				t.Errorf("Weights don't sum to 1.0: got %f", sum)
			}

			// All weights must be non-negative
			if w00 < 0 || w01 < 0 || w10 < 0 || w11 < 0 {
				t.Errorf("Negative weight: w00=%f, w01=%f, w10=%f, w11=%f",
					w00, w01, w10, w11)
			}
		})
	}
}

// TestBilinearWeightsExact tests exact corner cases
func TestBilinearWeightsExact(t *testing.T) {
	// At exact integer indices, one weight should be 1.0
	w00, w01, w10, w11 := BilinearWeights(2.0, 3.0)
	if w00 != 1.0 || w01 != 0.0 || w10 != 0.0 || w11 != 0.0 {
		t.Errorf("Expected w00=1.0 at exact index, got w00=%f, w01=%f, w10=%f, w11=%f",
			w00, w01, w10, w11)
	}

	// At center (0.5, 0.5), all weights should be 0.25
	w00, w01, w10, w11 = BilinearWeights(2.5, 3.5)
	if math.Abs(w00-0.25) > 1e-10 || math.Abs(w01-0.25) > 1e-10 ||
		math.Abs(w10-0.25) > 1e-10 || math.Abs(w11-0.25) > 1e-10 {
		t.Errorf("Expected all weights=0.25 at center, got w00=%f, w01=%f, w10=%f, w11=%f",
			w00, w01, w10, w11)
	}
}

// TestSub tests vector subtraction
func TestSub(t *testing.T) {
	x, y, z := Sub(5, 3, 7, 2, 1, 4)
	if x != 3 || y != 2 || z != 3 {
		t.Errorf("Sub(5,3,7, 2,1,4) = (%f,%f,%f), want (3,2,3)", x, y, z)
	}
}

// TestLength tests vector length calculation
func TestLength(t *testing.T) {
	tests := []struct {
		name    string
		x, y, z float64
		want    float64
	}{
		{"Unit X", 1, 0, 0, 1.0},
		{"Unit Y", 0, 1, 0, 1.0},
		{"Unit Z", 0, 0, 1, 1.0},
		{"3-4-5 triangle", 3, 4, 0, 5.0},
		{"Zero", 0, 0, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Length(tt.x, tt.y, tt.z)
			if math.Abs(got-tt.want) > 1e-10 {
				t.Errorf("Length(%f,%f,%f) = %f, want %f",
					tt.x, tt.y, tt.z, got, tt.want)
			}
		})
	}
}

// TestDistance tests distance calculation
func TestDistance(t *testing.T) {
	dist := Distance(0, 0, 0, 3, 4, 0)
	if math.Abs(dist-5.0) > 1e-10 {
		t.Errorf("Distance = %f, want 5.0", dist)
	}
}

// TestRotate tests vector rotation
func TestRotate(t *testing.T) {
	tests := []struct {
		name                string
		x, y, z             float64
		rx, ry, rz          float64
		wantX, wantY, wantZ float64
	}{
		{"No rotation", 1, 0, 0, 0, 0, 0, 1, 0, 0},
		{"90° Z rotation", 1, 0, 0, 0, 0, math.Pi / 2, 0, 1, 0},
		{"90° Y rotation", 1, 0, 0, 0, math.Pi / 2, 0, 0, 0, -1},
		{"90° X rotation", 0, 1, 0, math.Pi / 2, 0, 0, 0, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, z := Rotate(tt.x, tt.y, tt.z, tt.rx, tt.ry, tt.rz)
			if math.Abs(x-tt.wantX) > 1e-10 || math.Abs(y-tt.wantY) > 1e-10 || math.Abs(z-tt.wantZ) > 1e-10 {
				t.Errorf("Rotate(%f,%f,%f, %f,%f,%f) = (%f,%f,%f), want (%f,%f,%f)",
					tt.x, tt.y, tt.z, tt.rx, tt.ry, tt.rz,
					x, y, z, tt.wantX, tt.wantY, tt.wantZ)
			}
		})
	}
}

// TestThetaPhi tests spherical coordinate conversion
func TestThetaPhi(t *testing.T) {
	tests := []struct {
		name               string
		vecX, vecY, vecZ   float64
		angX, angY, angZ   float64
		wantTheta, wantPhi float64
	}{
		{"On-axis", 1, 0, 0, 0, 0, 0, 0, 0},
		{"Up 45°", 1, 0, 1, 0, 0, 0, math.Pi / 4, 0},
		{"Right 45°", 1, 1, 0, 0, 0, 0, 0, math.Pi / 4},
		{"Zero vector", 0, 0, 0, 0, 0, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theta, phi := ThetaPhi(tt.vecX, tt.vecY, tt.vecZ, tt.angX, tt.angY, tt.angZ)
			if math.Abs(theta-tt.wantTheta) > 1e-6 || math.Abs(phi-tt.wantPhi) > 1e-6 {
				t.Errorf("ThetaPhi = (%f,%f), want (%f,%f)",
					theta, phi, tt.wantTheta, tt.wantPhi)
			}
		})
	}
}

func TestDirectionToGLLAngles(t *testing.T) {
	tests := []struct {
		name                       string
		vecX, vecY, vecZ           float64
		angX, angY, angZ           float64
		wantMeridian, wantParallel float64
	}{
		{"Front", 1, 0, 0, 0, 0, 0, 0, 0},
		{"Top", 0, 0, 1, 0, 0, 0, 0, 90},
		{"Right", 0, 1, 0, 0, 0, 0, 90, 90},
		{"Back", -1, 0, 0, 0, 0, 0, 0, 180},
		{"Rotated source points to +Y", 0, 1, 0, 0, 0, math.Pi / 2, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meridian, parallel := DirectionToGLLAngles(
				tt.vecX,
				tt.vecY,
				tt.vecZ,
				tt.angX,
				tt.angY,
				tt.angZ,
			)
			if math.Abs(meridian-tt.wantMeridian) > 1e-6 || math.Abs(parallel-tt.wantParallel) > 1e-6 {
				t.Errorf(
					"DirectionToGLLAngles = (%f,%f), want (%f,%f)",
					meridian,
					parallel,
					tt.wantMeridian,
					tt.wantParallel,
				)
			}
		})
	}
}

// TestToComplexFromComplex tests round-trip conversion
func TestToComplexFromComplex(t *testing.T) {
	// Test data: level in dB, phase in radians
	level := []float64{0, -3, -6, -10}
	phase := []float64{0, math.Pi / 4, math.Pi / 2, math.Pi}

	// Convert to complex
	realPart, imagPart := ToComplex(level, phase)

	// Convert back
	newLevel, newPhase := FromComplex(realPart, imagPart)

	// Check round-trip accuracy
	for i := range level {
		if math.Abs(newLevel[i]-level[i]) > 1e-6 {
			t.Errorf("Round-trip level[%d]: got %f, want %f", i, newLevel[i], level[i])
		}
		if math.Abs(newPhase[i]-phase[i]) > 1e-6 {
			t.Errorf("Round-trip phase[%d]: got %f, want %f", i, newPhase[i], phase[i])
		}
	}
}

// TestToComplex tests magnitude calculation
func TestToComplex(t *testing.T) {
	// 0 dB = magnitude 1.0
	level := []float64{0}
	phase := []float64{0}
	realPart, imagPart := ToComplex(level, phase)

	expectedMag := 1.0
	gotMag := math.Sqrt(realPart[0]*realPart[0] + imagPart[0]*imagPart[0])
	if math.Abs(gotMag-expectedMag) > 1e-10 {
		t.Errorf("Magnitude at 0dB: got %f, want %f", gotMag, expectedMag)
	}

	// -6 dB ≈ magnitude 0.5
	level = []float64{-6.0206}
	phase = []float64{0}
	realPart, imagPart = ToComplex(level, phase)

	expectedMag = 0.5
	gotMag = math.Sqrt(realPart[0]*realPart[0] + imagPart[0]*imagPart[0])
	if math.Abs(gotMag-expectedMag) > 0.001 {
		t.Errorf("Magnitude at -6dB: got %f, want %f", gotMag, expectedMag)
	}
}

// TestAddComplexInPlace tests coherent summation
func TestAddComplexInPlace(t *testing.T) {
	// Two identical signals: sum should be +6dB
	level1 := []float64{0}
	phase1 := []float64{0}
	level2 := []float64{0}
	phase2 := []float64{0}

	AddComplexInPlace(level1, phase1, level2, phase2)

	expectedLevel := 20.0 * math.Log10(2.0) // ≈ 6.02 dB
	if math.Abs(level1[0]-expectedLevel) > 0.01 {
		t.Errorf("Coherent sum: got %f dB, want %f dB", level1[0], expectedLevel)
	}
}

// TestAddComplexInPlaceDestructive tests destructive interference
func TestAddComplexInPlaceDestructive(t *testing.T) {
	// Two signals 180° out of phase: should cancel
	level1 := []float64{0}
	phase1 := []float64{0}
	level2 := []float64{0}
	phase2 := []float64{math.Pi}

	AddComplexInPlace(level1, phase1, level2, phase2)

	// Should be very low level (near -∞ dB)
	if len(level1) == 0 {
		t.Error("level1 slice is empty after AddComplexInPlace")
	} else if level1[0] > -100 {
		t.Errorf("Destructive interference: got %f dB, want < -100 dB", level1[0])
	}
}

// TestMultiplyLevelPhase tests filter application
func TestMultiplyLevelPhase(t *testing.T) {
	level := []float64{0, -3, -6}
	phase := []float64{0, 0, 0}
	filterLevel := []float64{-3, -3, -3}
	filterPhase := []float64{math.Pi / 4, math.Pi / 4, math.Pi / 4}

	MultiplyLevelPhase(level, phase, filterLevel, filterPhase)

	// Check level addition
	expected := []float64{-3, -6, -9}
	for i := range level {
		if math.Abs(level[i]-expected[i]) > 1e-10 {
			t.Errorf("Level[%d] = %f, want %f", i, level[i], expected[i])
		}
	}

	// Check phase addition
	for i := range phase {
		if math.Abs(phase[i]-math.Pi/4) > 1e-10 {
			t.Errorf("Phase[%d] = %f, want %f", i, phase[i], math.Pi/4)
		}
	}
}

// TestAddGain tests gain addition
func TestAddGain(t *testing.T) {
	level := []float64{0, -3, -6}
	AddGain(level, 3)

	expected := []float64{3, 0, -3}
	for i := range level {
		if math.Abs(level[i]-expected[i]) > 1e-10 {
			t.Errorf("Level[%d] = %f, want %f", i, level[i], expected[i])
		}
	}
}

// TestAddDelay tests phase shift from time delay
func TestAddDelay(t *testing.T) {
	phase := []float64{0, 0, 0}
	freqs := []float64{100, 1000, 10000}
	delay := 0.001 // 1 ms

	AddDelay(phase, freqs, delay)

	// At 1 kHz, 1ms delay = 360° = 2π radians
	expectedPhase := 2.0 * math.Pi * 1000 * 0.001
	if math.Abs(phase[1]-expectedPhase) > 1e-6 {
		t.Errorf("Phase shift at 1kHz: got %f, want %f", phase[1], expectedPhase)
	}
}
