package xgll

import (
	"strings"
	"testing"
)

func TestValidateSystemConstraintsLA(t *testing.T) {
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"LA\", \"sys\", \"LA\"",
		"\"Layout\"",
		"\"Data\"",
		"\"Frames\", 1",
		"\"Connectors\", 1",
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	diags := ValidateSystemConstraints(doc)
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %d", len(diags))
	}
}

func TestValidateSystemConstraintsMissing(t *testing.T) {
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"LA\", \"sys\", \"LA\"",
		"\"Data\"",
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	diags := ValidateSystemConstraints(doc)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics for missing blocks")
	}
}

func TestValidateSystemConstraintsCL(t *testing.T) {
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"CL\", \"sys\", \"CL\"",
		"\"Layout\"",
		"\"Data\"",
		"\"Setups\", 1",
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	diags := ValidateSystemConstraints(doc)
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %d", len(diags))
	}
}

func TestValidateSystemConstraintsLSWarnings(t *testing.T) {
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"LS\", \"sys\", \"LS\"",
		"\"Layout\"",
		"\"Data\"",
		"\"Frames\", 1",
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	diags := ValidateSystemConstraints(doc)
	if len(diags) == 0 {
		t.Fatalf("expected warning diagnostics")
	}

	if diags[0].Severity != SeverityWarning {
		t.Fatalf("expected warning severity")
	}
}

func TestValidateDataConstraintsCounts(t *testing.T) {
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"LS\", \"sys\", \"LS\"",
		"\"Layout\"",
		"\"Data\"",
		"\"BoxTypes\", 1",
		"\"BoxType\", \"Box\", \"bx1\"",
		"\"Sources\", 2",
		"\"Source\", \"A\", \"srcA\", 0,0,0,0,0,0, \"SD_A\"",
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	diags := ValidateDataConstraints(doc)
	if len(diags) == 0 {
		t.Fatalf("expected count diagnostics")
	}
}

func TestValidateDataConstraintsRefs(t *testing.T) {
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"LS\", \"sys\", \"LS\"",
		"\"Layout\"",
		"\"Data\"",
		"\"BoxTypes\", 1",
		"\"BoxType\", \"Box\", \"bx1\"",
		"\"Sources\", 1",
		"\"Source\", \"A\", \"srcA\", 0,0,0,0,0,0, \"SD_A\"",
		"\"SourceDefinitions\", 1",
		"\"SourceDefinition\", \"SD_OK\"",
		"\"Links\", 1",
		"\"Link\", \"srcMissing\", \"FG_Missing\"",
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	diags := ValidateDataConstraints(doc)
	if len(diags) == 0 {
		t.Fatalf("expected reference diagnostics")
	}
}
