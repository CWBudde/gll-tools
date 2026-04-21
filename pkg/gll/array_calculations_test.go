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
	source := &SourceDefinition{BalloonData: testUniformBalloon(0)}
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
		if got := resp.Level[0]; math.Abs(got-30) > 1e-6 {
			t.Fatalf("level = %f, want 30.000000", got)
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

func testUniformBalloon(level float64) *BalloonData {
	def := LogSpectrumDefinition{
		BandsPerOctave: 1,
		StartFreq:      1000,
		PointCount:     1,
	}

	responses := make([]TransferFunction, 6)
	for i := range responses {
		responses[i] = TransferFunction{
			Definition: def,
			Level:      []float64{level},
			Phase:      []float64{0},
		}
	}

	return &BalloonData{
		AngularResolution: ResolutionDescriptor{
			Symmetry:     int32(SymmetryNone),
			MeridianStep: 90,
			ParallelStep: 90,
		},
		ResponseCount: int32(len(responses)),
		Responses:     responses,
	}
}

func testDirectionalBalloon() *BalloonData {
	balloon := testUniformBalloon(0)
	balloon.Responses[0].Level[0] = 10 // Front pole (parallel 0)
	balloon.Responses[1].Level[0] = 30 // Meridian 0, parallel 90 (top)
	balloon.Responses[2].Level[0] = 40 // Back pole (parallel 180)
	balloon.Responses[3].Level[0] = 20 // Meridian 90, parallel 90 (right)
	balloon.Responses[4].Level[0] = 60 // Meridian 180, parallel 90 (bottom)
	balloon.Responses[5].Level[0] = 50 // Meridian 270, parallel 90 (left)
	return balloon
}

func testRotationBalloon() *BalloonData {
	balloon := testUniformBalloon(0)
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
	switch symmetry {
	case SymmetryVertical, SymmetryHorizontal:
		levels = []float64{10, 20, 30, 40, 50}
	case SymmetryQuarter:
		levels = []float64{10, 20, 30, 40}
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
		ResponseCount: int32(len(responses)),
		Responses:     responses,
	}
}
