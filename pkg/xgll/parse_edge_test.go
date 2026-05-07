package xgll

import (
	"strings"
	"testing"
)

func TestParseEmptyInput(t *testing.T) {
	doc, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if len(doc.Statements) != 0 {
		t.Errorf("expected 0 statements for empty input, got %d", len(doc.Statements))
	}
}

func TestParseMissingGLLHeader(t *testing.T) {
	input := "\"Format\", \"3D\"\n\"System\", \"Test\", \"sysTest\", \"LS\""
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error when GLL header is missing")
	}
}

func TestParseValidMinimal(t *testing.T) {
	input := strings.Join([]string{
		`"GLL"`,
		`"Format", "3D"`,
		`"FormatVersion", "1.0"`,
		`"System", "Test", "sysTest", "LS"`,
		`"Layout"`,
		`"Data"`,
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("expected valid parse, got error: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil document")
	}
	if len(doc.Statements) == 0 {
		t.Fatal("expected at least one statement")
	}
}

func TestParseNumericArguments(t *testing.T) {
	input := strings.Join([]string{
		`"GLL"`,
		`"Format", "3D"`,
		`"FormatVersion", "1.0"`,
		`"System", "Test", "sysTest", "LS"`,
		`"SystemVersion", "1"`,
		`"Layout"`,
		`"BackgroundColor", 0.2, 0.3, 0.4`,
		`"Data"`,
		`"BoxTypes", 0`,
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil document")
	}

	// Find BackgroundColor statement and check numeric args
	found := false
	for _, stmt := range doc.Statements {
		if stmt.Keyword == "BackgroundColor" {
			found = true
			if len(stmt.Args) != 3 {
				t.Errorf("BackgroundColor: expected 3 args, got %d", len(stmt.Args))
			}
			for i, arg := range stmt.Args {
				if arg.Kind != ValueNumber {
					t.Errorf("BackgroundColor arg[%d]: expected ValueNumber, got %v", i, arg.Kind)
				}
			}
		}
	}
	if !found {
		t.Error("BackgroundColor statement not found")
	}
}

func TestParseStringArguments(t *testing.T) {
	input := strings.Join([]string{
		`"GLL"`,
		`"Format", "3D"`,
		`"FormatVersion", "1.0"`,
		`"System", "My System", "sysMySystem", "LS"`,
		`"Layout"`,
		`"Data"`,
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	for _, stmt := range doc.Statements {
		if stmt.Keyword == "System" {
			if len(stmt.Args) < 3 {
				t.Errorf("System: expected at least 3 args, got %d", len(stmt.Args))
				break
			}
			if stmt.Args[0].Kind != ValueString {
				t.Errorf("System arg[0]: expected ValueString, got %v", stmt.Args[0].Kind)
			}
			if stmt.Args[0].Raw != "My System" {
				t.Errorf("System arg[0].Raw = %q, want %q", stmt.Args[0].Raw, "My System")
			}
			break
		}
	}
}

func TestParseCommentLines(t *testing.T) {
	input := strings.Join([]string{
		`"GLL"`,
		`";", "This is a comment"`,
		`"Format", "3D"`,
		`"FormatVersion", "1.0"`,
		`"System", "Test", "sysTest", "LS"`,
		`"Layout"`,
		`"Data"`,
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil document")
	}
}

func TestParseDiagnosticsOnInvalidBlock(t *testing.T) {
	// Data block before Layout is invalid
	input := strings.Join([]string{
		`"GLL"`,
		`"Format", "3D"`,
		`"FormatVersion", "1.0"`,
		`"System", "Test", "sysTest", "LS"`,
		`"Data"`,
		`"Layout"`,
	}, "\n")

	doc, _ := Parse(strings.NewReader(input))
	if doc == nil {
		t.Fatal("expected document even with errors")
	}
	if !hasErrors(doc.Diagnostics) {
		t.Error("expected diagnostics for invalid block order")
	}
}

func TestParseNegativeNumbers(t *testing.T) {
	input := strings.Join([]string{
		`"GLL"`,
		`"Format", "3D"`,
		`"FormatVersion", "1.0"`,
		`"System", "Test", "sysTest", "LS"`,
		`"Layout"`,
		`"Data"`,
		`"BoxTypes", 1`,
		`"BoxType", "Cabinet", "bx1"`,
		`"Sources", 1`,
		`"Source", "S1", "src1", -0.05, 0, 0.15, 0, 0, 0, "SD_1"`,
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil document")
	}

	for _, stmt := range doc.Statements {
		if stmt.Keyword == "Source" && len(stmt.Args) >= 4 {
			posX := stmt.Args[2]
			if posX.Kind != ValueNumber {
				t.Errorf("Source X position: expected ValueNumber, got %v", posX.Kind)
			}
			if posX.Num != -0.05 {
				t.Errorf("Source X position: got %v, want -0.05", posX.Num)
			}
			break
		}
	}
}

func TestParseMultilineInput(t *testing.T) {
	input := strings.Join([]string{
		`"GLL"`,
		`"Format", "3D"`,
		`"FormatVersion", "1.0"`,
		`"System", "Test", "sysTest", "LS"`,
		`"Layout"`,
		`"Data"`,
		`"BoxTypes", 2`,
		`"BoxType", "Box A", "bxA"`,
		`"Sources", 0`,
		`"BoxType", "Box B", "bxB"`,
		`"Sources", 0`,
		`"SourceDefinitions", 0`,
		`"FilterGroups", 0`,
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil document")
	}

	boxTypeCount := 0
	for _, stmt := range doc.Statements {
		if stmt.Keyword == "BoxType" {
			boxTypeCount++
		}
	}
	if boxTypeCount != 2 {
		t.Errorf("expected 2 BoxType statements, got %d", boxTypeCount)
	}
}

func TestParseBlockTypes(t *testing.T) {
	input := strings.Join([]string{
		`"GLL"`,
		`"Format", "3D"`,
		`"FormatVersion", "1.0"`,
		`"System", "Test", "sysTest", "LS"`,
		`"Layout"`,
		`"Data"`,
		`"BoxTypes", 0`,
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	blockTypes := make(map[BlockType]bool)
	for _, b := range doc.Blocks {
		blockTypes[b.Type] = true
	}

	if !blockTypes[BlockHeader] {
		t.Error("expected BlockHeader block")
	}
	if !blockTypes[BlockData] {
		t.Error("expected BlockData block")
	}
}
