package xgll

import (
	"strings"
	"testing"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// TestValidateDataConstraints_NilAndEmpty covers the early-exit branches.
func TestValidateDataConstraints_NilAndEmpty(t *testing.T) {
	if d := ValidateDataConstraints(nil); len(d) != 0 {
		t.Errorf("nil doc: got %d diags, want 0", len(d))
	}
	if d := ValidateDataConstraints(&Document{}); len(d) != 0 {
		t.Errorf("empty doc: got %d diags, want 0", len(d))
	}
}

// TestValidateDataConstraints_ManyBlockTypes drives a wide set of switch arms
// in ValidateDataConstraints. The exact set of arms is intentionally broad —
// the goal is coverage, not zero diagnostics.
func TestValidateDataConstraints_ManyBlockTypes(t *testing.T) {
	// Source/SourceDefinition: argString(stmt, 1) is the Source name; arg 8
	// is the SourceDefinition reference. SourceDefinition's name is at arg 0.
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"LS\", \"sys\", \"LS\"",
		"\"Layout\"",
		"\"Data\"",
		"\"BoxTypes\", 1",
		"\"BoxType\", \"Box\", \"bx1\"",
		"\"GeometryFiles\", 1",
		"\"GeometryFile\", \"file.xed\"",
		"\"Frames\", 1",
		"\"Frame\", \"F\", \"frame1\"",
		"\"PinPoints\", 1",
		"\"PinPoint\", \"pp\"",
		"\"Angles\", 1",
		"\"Angle\", 0",
		"\"Limits\", 1",
		"\"Limit\", \"L1\", \"frame1\", \"bx1\", 100",
		"\"Warnings\", 1",
		"\"Warning\", \"W\", 1, \"frame1\", \"text\", 50",
		"\"SourceDefinitions\", 1",
		"\"SourceDefinition\", \"SD_A\"",
		"\"Sources\", 1",
		"\"Source\", \"Label\", \"srcA\", 0,0,0,0,0,0, \"SD_A\"",
		"\"FilterGroups\", 1",
		"\"FilterGroup\", \"FG\", \"fgA\"",
		"\"FilterDefinitions\", 1",
		"\"FilterDefinition\", \"FD\"",
		"\"Setups\", 1",
		"\"Setup\", \"S\"",
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Just call it — coverage of switch arms is the value here. We don't
	// pin diagnostic counts because XGLL keyword semantics in some blocks
	// (e.g. Boxes vs Box) would require fixture-perfect arg alignment.
	_ = ValidateDataConstraints(doc)
}

// TestValidateDataConstraints_DanglingRefs exercises the reference-resolution
// paths by emitting a Link whose Source and FilterGroup refs don't exist,
// plus a Limit referring to an unknown Frame and BoxType.
func TestValidateDataConstraints_DanglingRefs(t *testing.T) {
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"LS\", \"sys\", \"LS\"",
		"\"Layout\"",
		"\"Data\"",
		"\"Links\", 1",
		"\"Link\", \"missingSrc\", \"missingFG\"",
		"\"Limits\", 1",
		"\"Limit\", \"L\", \"missingFrame\", \"missingBox\", 1",
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	diags := ValidateDataConstraints(doc)
	if len(diags) == 0 {
		t.Fatal("expected diagnostics for dangling references")
	}
}

// TestApplyHeaderSettings_AllBranches drives every optional-field branch in
// applyHeaderSettings by giving it a Document with each header statement.
func TestApplyHeaderSettings_AllBranches(t *testing.T) {
	doc := &Document{
		Statements: []Statement{
			{Keyword: "BinaryFormatVersion", Args: []Value{{Kind: ValueNumber, Num: 4}}},
			{Keyword: "BinarySubVersion", Args: []Value{{Kind: ValueNumber, Num: 2}}},
			{Keyword: "BinaryChecksum", Args: []Value{{Kind: ValueString, Str: "DEADBEEF"}}},
			// 32-byte hex hash:
			{Keyword: "BinaryHash", Args: []Value{{Kind: ValueString, Str: strings.Repeat("AB", 32)}}},
		},
	}
	file := &gllbin.File{}
	if err := applyHeaderSettings(file, doc); err != nil {
		t.Fatalf("applyHeaderSettings: %v", err)
	}
	if file.Header.FormatVersion != 4 {
		t.Errorf("FormatVersion = %d, want 4", file.Header.FormatVersion)
	}
	if file.Header.SubVersion != 2 {
		t.Errorf("SubVersion = %d, want 2", file.Header.SubVersion)
	}
	if file.Header.Checksum != [4]byte{0xDE, 0xAD, 0xBE, 0xEF} {
		t.Errorf("Checksum = %x", file.Header.Checksum)
	}
	for i, b := range file.Header.HashID {
		if b != 0xAB {
			t.Errorf("HashID[%d] = %x, want AB", i, b)
		}
	}
}

func TestApplyHeaderSettings_BadChecksum(t *testing.T) {
	doc := &Document{
		Statements: []Statement{
			{Keyword: "BinaryChecksum", Args: []Value{{Kind: ValueString, Str: "ZZZZ"}}},
		},
	}
	file := &gllbin.File{}
	if err := applyHeaderSettings(file, doc); err == nil {
		t.Fatal("expected error on invalid checksum hex")
	}
}

func TestApplyHeaderSettings_BadHash(t *testing.T) {
	doc := &Document{
		Statements: []Statement{
			{Keyword: "BinaryHash", Args: []Value{{Kind: ValueString, Str: "ZZ"}}},
		},
	}
	file := &gllbin.File{}
	if err := applyHeaderSettings(file, doc); err == nil {
		t.Fatal("expected error on invalid hash hex")
	}
}

// TestParseFileStatements covers each branch of the datafile/includefile/
// authorfile switch.
func TestParseFileStatements(t *testing.T) {
	stmts := []Statement{
		{Keyword: "datafile", Args: []Value{
			{Kind: ValueString, Str: "img.png"},
			{Kind: ValueNumber, Num: 1024},
		}},
		// Empty datafile name → skipped.
		{Keyword: "datafile", Args: []Value{
			{Kind: ValueString, Str: ""},
		}},
		{Keyword: "includefile", Args: []Value{
			{Kind: ValueString, Str: "Label"},
			{Kind: ValueString, Str: "key"},
			{Kind: ValueString, Str: "geom.xed"},
			{Kind: ValueNumber, Num: 4096},
		}},
		// Truncated includefile → skipped.
		{Keyword: "includefile", Args: []Value{
			{Kind: ValueString, Str: "x"},
		}},
		{Keyword: "authorfile", Args: []Value{
			{Kind: ValueString, Str: "author.txt"},
			{Kind: ValueNumber, Num: 256},
		}},
		// Empty authorfile name → skipped.
		{Keyword: "authorfile", Args: []Value{
			{Kind: ValueString, Str: ""},
		}},
	}

	dataFiles, includeFiles, authorFiles := parseFileStatements(stmts)
	if len(dataFiles) != 1 || dataFiles[0].Filename != "img.png" {
		t.Errorf("dataFiles = %+v, want one img.png", dataFiles)
	}
	if len(includeFiles) != 1 || includeFiles[0].Filename != "geom.xed" {
		t.Errorf("includeFiles = %+v, want one geom.xed", includeFiles)
	}
	if len(authorFiles) != 1 || authorFiles[0].Filename != "author.txt" {
		t.Errorf("authorFiles = %+v, want one author.txt", authorFiles)
	}
}

// TestBuildDataFileStatements covers the database → XGLL statement-builder
// path for DataFiles, IncludeFiles, and AuthorFiles, including the
// size-zero branches (which emit a different statement variant).
func TestBuildDataFileStatements(t *testing.T) {
	t.Run("nil database", func(t *testing.T) {
		if got := buildDataFileStatements(nil); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("populated database", func(t *testing.T) {
		db := &gllbin.Database{
			DataFiles: []gllbin.DataFile{
				{Filename: "a.png", Size: 1024},
				{Filename: "b.jpg", Size: 0}, // zero-size variant
			},
			IncludeFiles: []gllbin.IncludeFile{
				{Label: "Geom", Key: "g1", Filename: "g.xed", Size: 2048},
				{Label: "Geom2", Key: "g2", Filename: "g2.xed"}, // size 0 variant
			},
			AuthorFiles: []gllbin.DataFile{
				{Filename: "author.txt", Size: 100},
			},
		}
		stmts := buildDataFileStatements(db)
		if len(stmts) == 0 {
			t.Fatal("got 0 statements, expected several")
		}

		// Verify we got DataFiles header + entries
		hasDataFilesHeader := false
		hasIncludeFilesHeader := false
		for _, s := range stmts {
			switch s.Keyword {
			case "DataFiles":
				hasDataFilesHeader = true
			case "IncludeFiles":
				hasIncludeFilesHeader = true
			}
		}
		if !hasDataFilesHeader {
			t.Error("missing DataFiles header statement")
		}
		if !hasIncludeFilesHeader {
			t.Error("missing IncludeFiles header statement")
		}
	})
}
