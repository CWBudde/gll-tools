package gll

import (
	"math"
	"testing"
)

func TestBalloonResponseAtGLLAngles(t *testing.T) {
	balloon := testDirectionalBalloon()

	tests := []struct {
		name               string
		meridian, parallel float64
		wantLevel          float64
	}{
		{"Front pole", 0, 0, 10},
		{"Top", 0, 90, 30},
		{"Right", 90, 90, 20},
		{"Back pole", 0, 180, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := balloon.responseAtGLLAngles(tt.meridian, tt.parallel)
			if resp == nil {
				t.Fatal("expected response, got nil")
			}
			if got := resp.Level[0]; math.Abs(got-tt.wantLevel) > 1e-6 {
				t.Fatalf("level = %f, want %f", got, tt.wantLevel)
			}
		})
	}
}

func TestComputeSystemResponseAtAppliesDelayOnce(t *testing.T) {
	source := &SourceDefinition{BalloonData: testUniformBalloon()}
	config := &ArrayConfig{
		Elements: []ArrayElement{
			{
				Position:   Vector3D{},
				SourceDefs: []*SourceDefinition{source},
			},
		},
	}

	resp := ComputeSystemResponseAt(
		config,
		Vector3D{X: 1},
		AirProperties{Speed: 343},
		false,
	)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	wantDelay := 1.0 / 343.0
	if math.Abs(resp.Delay-wantDelay) > 1e-9 {
		t.Fatalf("delay = %f, want %f", resp.Delay, wantDelay)
	}
}

func TestComputeSystemResponseAtUsesRotatedGLLAngles(t *testing.T) {
	source := &SourceDefinition{BalloonData: testRotationBalloon()}
	config := &ArrayConfig{
		Elements: []ArrayElement{
			{
				Position:   Vector3D{},
				Angles:     Vector3D{Z: math.Pi / 2},
				SourceDefs: []*SourceDefinition{source},
			},
		},
	}

	resp := ComputeSystemResponseAt(
		config,
		Vector3D{Y: 1},
		AirProperties{Speed: 343},
		false,
	)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if got := resp.Level[0]; math.Abs(got) > 1e-6 {
		t.Fatalf("level = %f, want 0.000000", got)
	}
}

func TestComputeSystemResponseAtUsesOrientationMatrix(t *testing.T) {
	source := &SourceDefinition{BalloonData: testRotationBalloon()}
	orientation := [9]float64{
		0, -1, 0,
		1, 0, 0,
		0, 0, 1,
	}
	config := &ArrayConfig{
		Elements: []ArrayElement{
			{
				Position:    Vector3D{},
				Orientation: &orientation,
				SourceDefs:  []*SourceDefinition{source},
			},
		},
	}

	resp := ComputeSystemResponseAt(
		config,
		Vector3D{Y: 1},
		AirProperties{Speed: 343},
		false,
	)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if got := resp.Level[0]; math.Abs(got) > 1e-6 {
		t.Fatalf("level = %f, want 0.000000", got)
	}
}

func TestComputeSystemResponseAtReceiverPlacementDirections(t *testing.T) {
	source := &SourceDefinition{BalloonData: testDirectionalBalloon()}
	config := &ArrayConfig{
		Elements: []ArrayElement{
			{
				Position:   Vector3D{},
				SourceDefs: []*SourceDefinition{source},
			},
		},
	}
	airProps := AirProperties{Speed: 343}

	tests := []struct {
		name      string
		receiver  Vector3D
		wantLevel float64
	}{
		{"front +X", Vector3D{X: 1}, 20},
		{"right +Y", Vector3D{Y: 1}, 30},
		{"top +Z", Vector3D{Z: 1}, 40},
		{"back -X", Vector3D{X: -1}, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ComputeSystemResponseAt(config, tt.receiver, airProps, false)
			if resp == nil {
				t.Fatal("expected response, got nil")
			}
			if got := resp.Level[0]; math.Abs(got-tt.wantLevel) > 1e-6 {
				t.Fatalf("level = %f, want %f", got, tt.wantLevel)
			}
		})
	}
}

func TestComputeSystemResponseAtShowsPathLengthInterference(t *testing.T) {
	def := LogSpectrumDefinition{
		BandsPerOctave: 1,
		StartFreq:      1000,
		PointCount:     2,
	}
	source := &SourceDefinition{BalloonData: testUniformBalloonWithDefinition(0, def)}
	halfWavelengthAt1k := 343.0 / (2.0 * 1000.0)
	config := &ArrayConfig{
		Elements: []ArrayElement{
			{
				Position:   Vector3D{},
				SourceDefs: []*SourceDefinition{source},
			},
			{
				Position:   Vector3D{X: halfWavelengthAt1k},
				SourceDefs: []*SourceDefinition{source},
			},
		},
	}

	resp := ComputeSystemResponseAt(
		config,
		Vector3D{X: 1000},
		AirProperties{Speed: 343},
		false,
	)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if len(resp.Level) != 2 {
		t.Fatalf("level len = %d, want 2", len(resp.Level))
	}

	if resp.Level[0] > resp.Level[1]-40 {
		t.Fatalf("expected 1 kHz half-wavelength null well below 2 kHz peak, got %.2f dB vs %.2f dB", resp.Level[0], resp.Level[1])
	}
}

func TestBalloonResponseAtGLLAnglesUsesParserSymmetryEnum(t *testing.T) {
	t.Run("vertical", func(t *testing.T) {
		balloon := testSymmetricBalloon(SymmetryVertical)
		resp := balloon.responseAtGLLAngles(270, 90)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
		if got := resp.Level[0]; math.Abs(got-40) > 1e-6 {
			t.Fatalf("level = %f, want 40.000000", got)
		}
	})

	t.Run("horizontal", func(t *testing.T) {
		balloon := testSymmetricBalloon(SymmetryHorizontal)
		resp := balloon.responseAtGLLAngles(0, 90)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
		if got := resp.Level[0]; math.Abs(got-40) > 1e-6 {
			t.Fatalf("level = %f, want 40.000000", got)
		}
	})

	t.Run("quarter", func(t *testing.T) {
		balloon := testSymmetricBalloon(SymmetryQuarter)
		resp := balloon.responseAtGLLAngles(225, 90)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
		want := pressureAverageDB(20, 40)
		if got := resp.Level[0]; math.Abs(got-want) > 1e-6 {
			t.Fatalf("level = %f, want %f", got, want)
		}
	})

	t.Run("axial", func(t *testing.T) {
		balloon := testSymmetricBalloon(SymmetryAxial)
		resp := balloon.responseAtGLLAngles(180, 90)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
		if got := resp.Level[0]; math.Abs(got-20) > 1e-6 {
			t.Fatalf("level = %f, want 20.000000", got)
		}
	})
}

func TestBalloonResponseAtGLLAnglesInterpolatesPhaseAcrossWrap(t *testing.T) {
	balloon := testUniformBalloon()
	balloon.Responses[1].Phase[0] = math.Pi - 0.1  // Meridian 0, parallel 90
	balloon.Responses[3].Phase[0] = -math.Pi + 0.1 // Meridian 90, parallel 90

	resp := balloon.responseAtGLLAngles(45, 90)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	got := math.Abs(resp.Phase[0])
	if math.Abs(got-math.Pi) > 0.11 {
		t.Fatalf("interpolated phase = %f, want near ±π", resp.Phase[0])
	}
}

func TestBalloonResponseAtGLLAnglesInterpolatesComplexPressure(t *testing.T) {
	balloon := testUniformBalloon()
	balloon.Responses[1].Phase[0] = 0       // Meridian 0, parallel 90
	balloon.Responses[3].Phase[0] = math.Pi // Meridian 90, parallel 90

	resp := balloon.responseAtGLLAngles(45, 90)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.Level[0] > -190 {
		t.Fatalf("interpolated level = %f, want near cancellation", resp.Level[0])
	}
}

func pressureAverageDB(levels ...float64) float64 {
	if len(levels) == 0 {
		return -200
	}
	sum := 0.0
	for _, level := range levels {
		sum += math.Pow(10, level/20.0)
	}
	avg := sum / float64(len(levels))
	if avg <= 0 {
		return -200
	}
	return 20 * math.Log10(avg)
}

func testUniformBalloon() *BalloonData {
	def := LogSpectrumDefinition{
		BandsPerOctave: 1,
		StartFreq:      1000,
		PointCount:     1,
	}

	return testUniformBalloonWithDefinition(0, def)
}

func testUniformBalloonWithDefinition(level float64, def LogSpectrumDefinition) *BalloonData {
	responses := make([]TransferFunction, 6)
	for i := range responses {
		responses[i] = TransferFunction{
			Definition: def,
			Level:      make([]float64, def.PointCount),
			Phase:      make([]float64, def.PointCount),
		}
		for j := range responses[i].Level {
			responses[i].Level[j] = level
		}
	}

	return &BalloonData{
		AngularResolution: ResolutionDescriptor{
			Symmetry:     int32(SymmetryNone),
			MeridianStep: 90,
			ParallelStep: 90,
		},
		ResponseCount: 6,
		Responses:     responses,
	}
}

func testDirectionalBalloon() *BalloonData {
	balloon := testUniformBalloon()
	balloon.Responses[0].Level[0] = 10 // Front pole (parallel 0)
	balloon.Responses[1].Level[0] = 30 // Meridian 0, parallel 90 (top)
	balloon.Responses[2].Level[0] = 40 // Back pole (parallel 180)
	balloon.Responses[3].Level[0] = 20 // Meridian 90, parallel 90 (right)
	balloon.Responses[4].Level[0] = 60 // Meridian 180, parallel 90 (bottom)
	balloon.Responses[5].Level[0] = 50 // Meridian 270, parallel 90 (left)
	return balloon
}

func testRotationBalloon() *BalloonData {
	balloon := testUniformBalloon()
	balloon.Responses[1].Level[0] = 5  // Meridian 0, parallel 90 (top)
	balloon.Responses[3].Level[0] = 10 // Meridian 90, parallel 90 (right)
	balloon.Responses[4].Level[0] = 15 // Meridian 180, parallel 90 (bottom)
	balloon.Responses[5].Level[0] = 20 // Meridian 270, parallel 90 (left)
	return balloon
}

func testSymmetricBalloon(symmetry SymmetryType) *BalloonData {
	def := LogSpectrumDefinition{
		BandsPerOctave: 1,
		StartFreq:      1000,
		PointCount:     1,
	}

	levels := []float64{10, 20, 30}
	responseCount := int32(3)
	switch symmetry {
	case SymmetryVertical, SymmetryHorizontal:
		levels = []float64{10, 20, 30, 40, 50}
		responseCount = 5
	case SymmetryQuarter:
		levels = []float64{10, 20, 30, 40}
		responseCount = 4
	}

	responses := make([]TransferFunction, len(levels))
	for i, level := range levels {
		responses[i] = TransferFunction{
			Definition: def,
			Level:      []float64{level},
			Phase:      []float64{0},
		}
	}

	return &BalloonData{
		AngularResolution: ResolutionDescriptor{
			Symmetry:     int32(symmetry),
			MeridianStep: 90,
			ParallelStep: 90,
		},
		ResponseCount: responseCount,
		Responses:     responses,
	}
}
