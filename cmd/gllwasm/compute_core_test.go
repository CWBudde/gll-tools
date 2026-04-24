package main

import (
	"math"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

func TestComputeArrayResponseForFileSyntheticDirections(t *testing.T) {
	file := syntheticVisualizationFile()
	req := ArrayResponseRequest{
		Elements: []ArrayElementInput{{SourceKey: "src"}},
		Receiver: ReceiverInput{Y: 1},
		AirProps: AirPropsInput{Speed: 343},
	}

	result := computeArrayResponseForFile(file, nil, req)
	if !result.Success {
		t.Fatalf("computeArrayResponseForFile failed: %s", result.Error)
	}
	if len(result.Frequencies) != 1 || result.Frequencies[0] != 1000 {
		t.Fatalf("frequencies = %v, want [1000]", result.Frequencies)
	}
	if got := result.Level[0]; math.Abs(got-10) > 1e-6 {
		t.Fatalf("right receiver level = %f, want 10", got)
	}
}

func TestComputeArrayResponseDataReportsParseErrors(t *testing.T) {
	result := computeArrayResponseData([]byte("not a gll"), "{}")
	if result.Success {
		t.Fatal("expected parse failure")
	}
	if result.Error == "" {
		t.Fatal("expected parse error message")
	}
	if got := marshalArrayResult(result); got == "" {
		t.Fatal("expected marshaled array result")
	}
}

func TestComputeArrayBalloonForFileProgressAndResponseParity(t *testing.T) {
	file := syntheticVisualizationFile()
	receivers := []ReceiverInput{
		{X: 1},
		{Y: 1},
		{Z: 1},
		{X: -1},
	}
	req := ArrayBalloonRequest{
		Elements:  []ArrayElementInput{{SourceKey: "src"}},
		Receivers: receivers,
		AirProps:  AirPropsInput{Speed: 343},
	}

	var progressEvents [][2]int
	result := computeArrayBalloonForFile(file, nil, req, func(completed, total int) {
		progressEvents = append(progressEvents, [2]int{completed, total})
	})
	if !result.Success {
		t.Fatalf("computeArrayBalloonForFile failed: %s", result.Error)
	}
	if len(result.Results) != len(receivers) {
		t.Fatalf("result count = %d, want %d", len(result.Results), len(receivers))
	}
	if len(progressEvents) == 0 {
		t.Fatal("expected progress events")
	}
	if progressEvents[0] != [2]int{0, len(receivers)} {
		t.Fatalf("first progress event = %v, want [0 %d]", progressEvents[0], len(receivers))
	}
	if progressEvents[len(progressEvents)-1] != [2]int{len(receivers), len(receivers)} {
		t.Fatalf("last progress event = %v, want [%d %d]", progressEvents[len(progressEvents)-1], len(receivers), len(receivers))
	}

	for i, receiver := range receivers {
		single := computeArrayResponseForFile(file, nil, ArrayResponseRequest{
			Elements: req.Elements,
			Receiver: receiver,
			AirProps: req.AirProps,
		})
		if !single.Success {
			t.Fatalf("single response %d failed: %s", i, single.Error)
		}
		if got, want := result.Results[i].Level[0], single.Level[0]; math.Abs(got-want) > 1e-6 {
			t.Fatalf("balloon response %d level = %f, want single response %f", i, got, want)
		}
	}
}

func TestComputeArrayBalloonDataReportsParseErrors(t *testing.T) {
	result := computeArrayBalloonData([]byte("not a gll"), "{}", nil)
	if result.Success {
		t.Fatal("expected parse failure")
	}
	if result.Error == "" {
		t.Fatal("expected parse error message")
	}
	if got := marshalBalloonResult(result); got == "" {
		t.Fatal("expected marshaled balloon result")
	}
	if got := marshalBalloonProgressEvent(ArrayBalloonProgressEvent{Type: "complete", Result: &result}); got == "" {
		t.Fatal("expected marshaled progress event")
	}
}

func syntheticVisualizationFile() *gll.File {
	def := gll.LogSpectrumDefinition{
		BandsPerOctave: 1,
		StartFreq:      1000,
		PointCount:     1,
	}
	levels := []float64{
		0,  // Front pole (+X)
		20, // Top (+Z)
		30, // Back (-X)
		10, // Right (+Y)
		40, // Bottom (-Z)
		50, // Left (-Y)
	}
	responses := make([]gll.TransferFunction, len(levels))
	for i, level := range levels {
		responses[i] = gll.TransferFunction{
			Definition: def,
			Level:      []float64{level},
			Phase:      []float64{0},
		}
	}

	return &gll.File{
		Database: &gll.Database{
			SourceDefinitions: []gll.SourceDefinitionItem{
				{
					Key: "src",
					Definition: &gll.SourceDefinition{
						BalloonData: &gll.BalloonData{
							AngularResolution: gll.ResolutionDescriptor{
								Symmetry:     int32(gll.SymmetryNone),
								MeridianStep: 90,
								ParallelStep: 90,
							},
							ResponseCount: 6,
							Responses:     responses,
						},
					},
				},
			},
		},
	}
}
