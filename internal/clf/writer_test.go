package clf

import (
	"bytes"
	"strings"
	"testing"
)

func TestWrite_header(t *testing.T) {
	var buf bytes.Buffer

	params := &ExportParams{
		CLFType:         2,
		ModelName:       "Test Speaker",
		Manufacturer:    "Test Corp",
		Description:     "A test speaker",
		Website:         "https://example.com",
		Weight:          12.5,
		MinBand:         100,
		MaxBand:         20000,
		Impedance:       8,
		Radiation:       "halfsphere",
		Symmetry:        "<none>",
		Reference:       "absolute",
		AzimuthCount:    2,
		PolarCount:      3,
		StepDeg:         5,
		BandFrequencies: []float64{1000},
		PolarLevels: [][][]float64{
			{{0}, {0}, {0}},
			{{0}, {0}, {0}},
		},
	}

	err := Write(&buf, params)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	output := buf.String()

	checks := []struct {
		desc string
		want string
	}{
		{"start tag", "<CLF2>"},
		{"model name", "<MODELNAME>\tTest Speaker"},
		{"manufacturer", "<MANUFACTURER>\tTest Corp"},
		{"description", "<DESCRIPTION>\tA test speaker"},
		{"website", "<WEB-SITE>\thttps://example.com"},
		{"impedance", "<IMPEDANCE>\t8"},
		{"radiation", "<RADIATION>\t<halfsphere>"},
		{"symmetry", "<BALLOON-SYMMETRY>\t<none>"},
		{"end tag", "<CLF2END>"},
		{"band data", "<BAND>\t1000"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.want) {
			t.Errorf("missing %s: want %q in output", c.desc, c.want)
		}
	}
}

func TestWrite_bandData(t *testing.T) {
	var buf bytes.Buffer

	params := &ExportParams{
		CLFType:         2,
		ModelName:       "X",
		Manufacturer:    "Y",
		MinBand:         100,
		MaxBand:         20000,
		Radiation:       "halfsphere",
		Symmetry:        "<none>",
		Reference:       "absolute",
		AzimuthCount:    2,
		PolarCount:      3,
		StepDeg:         5,
		BandFrequencies: []float64{1000},
		PolarLevels: [][][]float64{
			{{90}, {85}, {80}},
			{{88}, {83}, {78}},
		},
	}

	err := Write(&buf, params)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(output, "\n")

	bandIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "<BAND>\t1000") {
			bandIdx = i
			break
		}
	}
	if bandIdx == -1 {
		t.Fatal("BAND line for 1000 Hz not found")
	}

	row0 := lines[bandIdx+1]
	if !strings.Contains(row0, "90.00") || !strings.Contains(row0, "85.00") || !strings.Contains(row0, "80.00") {
		t.Errorf("unexpected row0: %s", row0)
	}
	row1 := lines[bandIdx+2]
	if !strings.Contains(row1, "88.00") || !strings.Contains(row1, "83.00") || !strings.Contains(row1, "78.00") {
		t.Errorf("unexpected row1: %s", row1)
	}
}

func TestWrite_cabinetDXF(t *testing.T) {
	var buf bytes.Buffer

	params := &ExportParams{
		CLFType:         2,
		ModelName:       "X",
		Manufacturer:    "Y",
		AzimuthCount:    1,
		PolarCount:      1,
		BandFrequencies: []float64{1000},
		PolarLevels:     [][][]float64{{{90}}},
		CabinetDXF:      "cabinet.dxf",
		CabinetSystem:   "My Cabinet",
	}

	err := Write(&buf, params)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<CABINET>\t<dxf>cabinet.dxf") {
		t.Error("missing CABINET DXF reference")
	}
	if !strings.Contains(output, "<CABINET-SYSTEM>\tMy Cabinet") {
		t.Error("missing CABINET-SYSTEM")
	}
}

func TestWrite_CLF1(t *testing.T) {
	var buf bytes.Buffer

	params := &ExportParams{
		CLFType:         1,
		ModelName:       "CLF1 Speaker",
		Manufacturer:    "Test",
		AzimuthCount:    1,
		PolarCount:      1,
		BandFrequencies: []float64{1000},
		PolarLevels:     [][][]float64{{{90}}},
	}

	err := Write(&buf, params)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	output := buf.String()
	if !strings.HasPrefix(output, "<CLF1>") {
		t.Error("expected CLF1 start tag")
	}
	if !strings.Contains(output, "<CLF1END>") {
		t.Error("expected CLF1END tag")
	}
}
