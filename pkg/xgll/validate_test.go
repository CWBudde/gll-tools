package xgll

import (
	"strings"
	"testing"
)

func TestValidateSystemConstraintsLA(t *testing.T) {
	// LA must include Frames and Connectors blocks
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

	// Parse input document
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Validate constraints
	diags := ValidateSystemConstraints(doc)
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %d", len(diags))
	}
}

func TestValidateSystemConstraintsMissing(t *testing.T) {
	// Missing blocks should trigger diagnostics
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"LA\", \"sys\", \"LA\"",
		"\"Data\"",
	}, "\n")

	// Parse input document
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Validate constraints
	diags := ValidateSystemConstraints(doc)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics for missing blocks")
	}
}

func TestValidateSystemConstraintsCL(t *testing.T) {
	// CL must include Setups block
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"CL\", \"sys\", \"CL\"",
		"\"Layout\"",
		"\"Data\"",
		"\"Setups\", 1",
	}, "\n")

	// Parse input document
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Validate constraints
	diags := ValidateSystemConstraints(doc)
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %d", len(diags))
	}
}

func TestValidateSystemConstraintsLSWarnings(t *testing.T) {
	// LS should warn for LA/CL-specific blocks
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"LS\", \"sys\", \"LS\"",
		"\"Layout\"",
		"\"Data\"",
		"\"Frames\", 1",
	}, "\n")

	// Parse input document
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Validate constraints
	diags := ValidateSystemConstraints(doc)
	if len(diags) == 0 {
		t.Fatalf("expected warning diagnostics")
	}

	// Ensure warning severity
	if diags[0].Severity != SeverityWarning {
		t.Fatalf("expected warning severity")
	}
}

func TestValidateDataConstraintsCounts(t *testing.T) {
	// Mismatched counts should trigger diagnostics
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

	// Parse input document
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Validate data constraints
	diags := ValidateDataConstraints(doc)
	if len(diags) == 0 {
		t.Fatalf("expected count diagnostics")
	}
}

func TestValidateDataConstraintsRefs(t *testing.T) {
	// Missing references should trigger diagnostics
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

	// Parse input document
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Validate data constraints
	diags := ValidateDataConstraints(doc)
	if len(diags) == 0 {
		t.Fatalf("expected reference diagnostics")
	}
}
