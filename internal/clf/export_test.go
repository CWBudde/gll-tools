package clf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

func TestExportSource_basic(t *testing.T) {
	src := &gll.SourceDefinition{
		Label:                "Test Source",
		CompanyLabel:         "Test Corp",
		Description:          "Test description",
		NominalBandwidthFrom: 100,
		NominalBandwidthTo:   20000,
		MeasuredVoltage:      2.828,
		MeasuredDistance:     1.0,
		RatedImpedance:       8,
		BalloonData: &gll.BalloonData{
			AngularResolution: gll.ResolutionDescriptor{
				Symmetry:     0,
				MeridianStep: 5,
				ParallelStep: 5,
			},
			ResponseCount: 2664,
			Responses:     makeTestResponses(72, 37),
		},
	}

	gen := &gll.GenSystem{
		Label:   "Test System",
		Company: "Test Corp",
	}

	var buf bytes.Buffer
	err := ExportSource(&buf, src, gen)
	if err != nil {
		t.Fatalf("ExportSource failed: %v", err)
	}

	output := buf.String()
	if !strings.HasPrefix(output, "<CLF2>") {
		t.Error("expected CLF2 format")
	}
	if !strings.Contains(output, "<MANUFACTURER>\tTest Corp") {
		t.Error("missing manufacturer")
	}
	if !strings.Contains(output, "<MODELNAME>\tTest Source") {
		t.Error("missing model name")
	}
	if !strings.Contains(output, "<IMPEDANCE>\t8") {
		t.Error("missing impedance")
	}

	bandCount := strings.Count(output, "<BAND>\t")
	if bandCount != 24 {
		t.Errorf("expected 24 bands, got %d", bandCount)
	}
}

func TestExportSource_nilSource(t *testing.T) {
	var buf bytes.Buffer
	err := ExportSource(&buf, nil, nil)
	if err == nil {
		t.Error("expected error for nil source")
	}
}

func TestExportSource_noBalloon(t *testing.T) {
	var buf bytes.Buffer
	err := ExportSource(&buf, &gll.SourceDefinition{}, nil)
	if err == nil {
		t.Error("expected error for nil balloon")
	}
}

func TestExportSource_noResponses(t *testing.T) {
	var buf bytes.Buffer
	src := &gll.SourceDefinition{
		BalloonData: &gll.BalloonData{},
	}
	err := ExportSource(&buf, src, nil)
	if err == nil {
		t.Error("expected error for empty responses")
	}
}

func TestExportSource_withBoxTypes(t *testing.T) {
	src := &gll.SourceDefinition{
		Label: "Driver A",
		BalloonData: &gll.BalloonData{
			AngularResolution: gll.ResolutionDescriptor{
				MeridianStep: 5,
				ParallelStep: 5,
			},
			Responses: makeTestResponses(72, 37),
		},
	}

	boxes := []gll.BoxType{
		{
			Label:  "Cabinet X",
			Weight: 15.5,
			SourcePlacements: []gll.BoxSource{
				{SourceDefKey: "src-a"},
			},
		},
		{
			Label:  "Cabinet Y",
			Weight: 20.0,
			SourcePlacements: []gll.BoxSource{
				{SourceDefKey: "src-b"},
			},
		},
	}

	var buf bytes.Buffer
	err := ExportSource(&buf, src, nil,
		WithBoxTypes(boxes),
		WithSourceKey("src-a"),
	)
	if err != nil {
		t.Fatalf("ExportSource failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<WEIGHT>\t15.5") {
		t.Error("expected weight 15.5 from box type")
	}
	if !strings.Contains(output, "<CABINET-SYSTEM>\tCabinet X") {
		t.Error("expected cabinet-system label from box type")
	}
}

func TestExportSource_withCabinetDXF(t *testing.T) {
	src := &gll.SourceDefinition{
		Label: "Driver A",
		BalloonData: &gll.BalloonData{
			AngularResolution: gll.ResolutionDescriptor{
				MeridianStep: 5,
				ParallelStep: 5,
			},
			Responses: makeTestResponses(72, 37),
		},
	}

	var buf bytes.Buffer
	err := ExportSource(&buf, src, nil,
		WithCabinetDXF("speaker.dxf"),
	)
	if err != nil {
		t.Fatalf("ExportSource failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<CABINET>\t<dxf>speaker.dxf") {
		t.Error("expected CABINET DXF reference")
	}
}

func makeTestResponses(merCount, parCount int) []gll.TransferFunction {
	total := merCount*parCount - (merCount-1)*2 // pole deduplication
	responses := make([]gll.TransferFunction, total)
	for i := range responses {
		responses[i] = gll.TransferFunction{
			Definition: gll.LogSpectrumDefinition{
				BandsPerOctave: 24,
				StartFreq:      20,
				PointCount:     240,
			},
			Level: make([]float64, 240),
			Phase: make([]float64, 240),
		}
		for j := range responses[i].Level {
			responses[i].Level[j] = 90.0 - float64(i%parCount)*0.5
		}
	}
	return responses
}
