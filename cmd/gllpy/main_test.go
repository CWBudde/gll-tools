package main

import (
	"math"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

func TestExpandBoxElementSourcesAppliesSourcePlacements(t *testing.T) {
	sourceDefMap := map[string]*gll.SourceDefinition{
		"src-a": {},
		"src-b": {},
	}
	box := &gll.BoxType{
		SourcePlacements: []gll.BoxSource{
			{
				SourceDefKey: "src-a",
				Position:     gll.Vector3D{X: 1000, Y: 2000, Z: -500},
			},
			{
				SourceDefKey: "src-b",
				Position:     gll.Vector3D{X: -500, Y: 0, Z: 250},
				Angles:       gll.Vector3D{Y: math.Pi / 8},
			},
		},
	}
	elem := ArrayElementJSON{
		Position: &Vector3DJSON{X: 1, Y: 2, Z: 3},
		Angles:   &Vector3DJSON{X: math.Pi / 4},
		Gain:     -3,
	}

	got := expandBoxElementSources(elem, box, sourceDefMap)
	if len(got) != 2 {
		t.Fatalf("expanded element count = %d, want 2", len(got))
	}

	assertVector(t, got[0].Position, gll.Vector3D{X: 3.121320343559643, Y: 2.707106781186548, Z: 2.5})
	assertVector(t, got[1].Position, gll.Vector3D{X: 0.6464466094067263, Y: 2.353553390593274, Z: 3.25})
	if got[0].Gain != -3 || got[1].Gain != -3 {
		t.Fatalf("expanded gains = %f/%f, want -3/-3", got[0].Gain, got[1].Gain)
	}
	if len(got[0].SourceDefs) != 1 || got[0].SourceDefs[0] != sourceDefMap["src-a"] {
		t.Fatal("first source definition not preserved")
	}
	if got[1].Angles.Y != math.Pi/8 {
		t.Fatalf("placement angle not added: y = %f", got[1].Angles.Y)
	}
	if got[1].Orientation == nil {
		t.Fatal("expected composed orientation matrix")
	}
}

func TestExpandBoxElementSourcesFallsBackToBoxSources(t *testing.T) {
	sourceDefMap := map[string]*gll.SourceDefinition{
		"src-a": {},
		"src-b": {},
	}
	box := &gll.BoxType{Sources: []string{"src-a", "missing", "src-b"}}
	elem := ArrayElementJSON{
		Position: &Vector3DJSON{X: 1, Y: 2, Z: 3},
		Angles:   &Vector3DJSON{Z: math.Pi / 2},
		Gain:     2,
	}

	got := expandBoxElementSources(elem, box, sourceDefMap)
	if len(got) != 2 {
		t.Fatalf("expanded element count = %d, want 2", len(got))
	}
	for _, element := range got {
		assertVector(t, element.Position, gll.Vector3D{X: 1, Y: 2, Z: 3})
		assertVector(t, element.Angles, gll.Vector3D{Z: math.Pi / 2})
		if element.Gain != 2 {
			t.Fatalf("gain = %f, want 2", element.Gain)
		}
		if element.Orientation == nil {
			t.Fatal("expected orientation matrix")
		}
	}
}

func TestExpandedBoxSourceOffsetsProducePathLengthInterference(t *testing.T) {
	def := gll.LogSpectrumDefinition{
		BandsPerOctave: 1,
		StartFreq:      1000,
		PointCount:     2,
	}
	sourceDefMap := map[string]*gll.SourceDefinition{
		"src-a": {BalloonData: testUniformBalloonWithDefinition(def)},
		"src-b": {BalloonData: testUniformBalloonWithDefinition(def)},
	}
	halfWavelengthAt1kM := 343.0 / (2.0 * 1000.0)
	box := &gll.BoxType{
		SourcePlacements: []gll.BoxSource{
			{SourceDefKey: "src-a"},
			{
				SourceDefKey: "src-b",
				Position:     gll.Vector3D{X: halfWavelengthAt1kM * 1000},
			},
		},
	}

	elements := expandBoxElementSources(ArrayElementJSON{}, box, sourceDefMap)
	if len(elements) != 2 {
		t.Fatalf("expanded element count = %d, want 2", len(elements))
	}

	resp := gll.ComputeSystemResponseAt(
		&gll.ArrayConfig{Elements: elements},
		gll.Vector3D{X: 1000},
		gll.AirProperties{Speed: 343},
		false,
	)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Level[0] > resp.Level[1]-40 {
		t.Fatalf("expected 1 kHz half-wavelength null well below 2 kHz peak, got %.2f dB vs %.2f dB", resp.Level[0], resp.Level[1])
	}
}

func testUniformBalloonWithDefinition(def gll.LogSpectrumDefinition) *gll.BalloonData {
	responses := make([]gll.TransferFunction, 6)
	for i := range responses {
		responses[i] = gll.TransferFunction{
			Definition: def,
			Level:      make([]float64, def.PointCount),
			Phase:      make([]float64, def.PointCount),
		}
	}

	return &gll.BalloonData{
		AngularResolution: gll.ResolutionDescriptor{
			Symmetry:     int32(gll.SymmetryNone),
			MeridianStep: 90,
			ParallelStep: 90,
		},
		ResponseCount: 6,
		Responses:     responses,
	}
}

func assertVector(t *testing.T, got, want gll.Vector3D) {
	t.Helper()
	const tol = 1e-9
	if math.Abs(got.X-want.X) > tol || math.Abs(got.Y-want.Y) > tol || math.Abs(got.Z-want.Z) > tol {
		t.Fatalf("vector = (%f, %f, %f), want (%f, %f, %f)", got.X, got.Y, got.Z, want.X, want.Y, want.Z)
	}
}
