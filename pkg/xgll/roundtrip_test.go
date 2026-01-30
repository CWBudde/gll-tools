package xgll

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

func TestRoundTripGLLViaXGLL(t *testing.T) {
	// Guard for expensive, byte-for-byte roundtrip test
	if os.Getenv("GLL_ROUNDTRIP") != "1" {
		t.Skip("set GLL_ROUNDTRIP=1 to run full byte-for-byte roundtrip tests")
	}

	// Find GLL fixtures
	paths, err := filepath.Glob(filepath.FromSlash("../../testdata/gll/*.gll"))
	if err != nil {
		t.Fatalf("glob gll: %v", err)
	}
	if len(paths) == 0 {
		t.Skip("no .gll files found in testdata/gll")
	}

	// Use temp output directory
	outDir := t.TempDir()

	for _, path := range paths {
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		t.Run(base, func(t *testing.T) {
			// Parse original GLL
			src, err := os.Open(path)
			if err != nil {
				t.Fatalf("open gll: %v", err)
			}
			defer src.Close()

			file, err := gllbin.Parse(src)
			if err != nil {
				t.Fatalf("parse gll: %v", err)
			}

			// Convert to XGLL
			doc, err := BuildXGLLDocument(file)
			if err != nil {
				t.Fatalf("build xgll: %v", err)
			}

			// Write XGLL to temp file
			xgllPath := filepath.Join(outDir, base+".xgll")
			if err := writeXGLLFile(xgllPath, doc); err != nil {
				t.Fatalf("write xgll: %v", err)
			}

			// Parse XGLL back into document
			roundDoc, err := ParseFile(xgllPath)
			if err != nil {
				t.Fatalf("parse xgll: %v", err)
			}

			// Write roundtripped GLL
			gllPath := filepath.Join(outDir, base+".gll")
			if err := writeGLLFile(gllPath, roundDoc); err != nil {
				t.Fatalf("write gll: %v", err)
			}

			// Load original and roundtripped bytes
			origBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read original: %v", err)
			}

			newBytes, err := os.ReadFile(gllPath)
			if err != nil {
				t.Fatalf("read roundtrip: %v", err)
			}

			// Compare byte-for-byte
			if !bytes.Equal(origBytes, newBytes) {
				roundFile, err := gllbin.Parse(bytes.NewReader(newBytes))
				if err != nil {
					t.Fatalf("roundtrip mismatch: orig=%s new=%s (len %d vs %d) parse new failed: %v", hashBytes(origBytes), hashBytes(newBytes), len(origBytes), len(newBytes), err)
				}

				t.Fatalf("roundtrip mismatch: orig=%s new=%s (len %d vs %d) header=%s gensystem=%s db=%s resources=%s",
					hashBytes(origBytes),
					hashBytes(newBytes),
					len(origBytes),
					len(newBytes),
					formatHeaderDiff(file.Header, roundFile.Header),
					formatGenSystemDiff(file.GenSystem, roundFile.GenSystem),
					formatDatabaseDiff(file.Database, roundFile.Database),
					formatResourceDiff(file.Resources, roundFile.Resources),
				)
			}
		})
	}
}

func TestRoundTripXGLLSystem(t *testing.T) {
	// Roundtrip XGLL -> GLL -> XGLL at a system level
	paths, err := filepath.Glob(filepath.FromSlash("../../testdata/xgll/*.xgll"))
	if err != nil {
		t.Fatalf("glob xgll: %v", err)
	}
	if len(paths) == 0 {
		t.Skip("no .xgll files found in testdata/xgll")
	}

	for _, path := range paths {
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		t.Run(base, func(t *testing.T) {
			// Parse XGLL input
			doc, err := ParseFile(path)
			if err != nil {
				t.Fatalf("parse xgll: %v", err)
			}

			// Convert to GLL
			var buf bytes.Buffer
			writer, err := GetWriter("gll")
			if err != nil {
				t.Fatalf("get gll writer: %v", err)
			}
			if err := writer.Write(doc, &buf); err != nil {
				t.Fatalf("write gll: %v", err)
			}

			// Parse GLL back
			file, err := gllbin.Parse(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("parse gll: %v", err)
			}

			// Convert back to XGLL
			roundDoc, err := BuildXGLLDocument(file)
			if err != nil {
				t.Fatalf("build xgll: %v", err)
			}

			// Compare a subset of GenSystem metadata
			origFile, err := BuildGLLFile(doc)
			if err != nil {
				t.Fatalf("build gll file: %v", err)
			}

			roundFile, err := BuildGLLFile(roundDoc)
			if err != nil {
				t.Fatalf("build gll file from roundtrip: %v", err)
			}

			if err := compareGenSystem(origFile.GenSystem, roundFile.GenSystem); err != nil {
				t.Fatalf("gensystem mismatch: %v", err)
			}
		})
	}
}

func writeXGLLFile(path string, doc *Document) error {
	// Write XGLL document to file
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	return WriteXGLL(doc, out)
}

func writeGLLFile(path string, doc *Document) error {
	// Write GLL document to file
	writer, err := GetWriter("gll")
	if err != nil {
		return err
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	return writer.Write(doc, out)
}

func compareGenSystem(a, b gllbin.GenSystem) error {
	// Compare a subset of GenSystem fields
	switch {
	case a.Label != b.Label:
		return fmtError("label", a.Label, b.Label)
	case a.Key != b.Key:
		return fmtError("key", a.Key, b.Key)
	case a.Type != b.Type:
		return fmtError("type", a.Type, b.Type)
	case a.Version != b.Version:
		return fmtError("version", a.Version, b.Version)
	case a.Company != b.Company:
		return fmtError("company", a.Company, b.Company)
	case a.InfoText != b.InfoText:
		return fmtError("info", a.InfoText, b.InfoText)
	case a.CopyrightText != b.CopyrightText:
		return fmtError("copyright", a.CopyrightText, b.CopyrightText)
	case a.SupportText != b.SupportText:
		return fmtError("support", a.SupportText, b.SupportText)
	case a.WebsiteText != b.WebsiteText:
		return fmtError("website", a.WebsiteText, b.WebsiteText)
	case a.EmailText != b.EmailText:
		return fmtError("email", a.EmailText, b.EmailText)
	case a.BackgroundColor != b.BackgroundColor:
		return fmtError("background_color", a.BackgroundColor, b.BackgroundColor)
	}

	return nil
}

func fmtError(field string, a, b any) error {
	// Helper for field mismatch formatting
	return &fieldMismatch{field: field, a: a, b: b}
}

type fieldMismatch struct {
	field string
	a     any
	b     any
}

func (m *fieldMismatch) Error() string {
	// Error implements error interface
	return fmt.Sprintf("field %s mismatch: %v != %v", m.field, m.a, m.b)
}

func hashBytes(data []byte) string {
	// Hash bytes for concise comparison output
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func formatHeaderDiff(a, b gllbin.Header) string {
	// Summarize header differences
	return fmt.Sprintf("fmt=%d/%d sub=%d/%d checksum=%t hash=%t",
		a.FormatVersion, b.FormatVersion,
		a.SubVersion, b.SubVersion,
		hasChecksum(a.Checksum) == hasChecksum(b.Checksum),
		hasHash(a.HashID) == hasHash(b.HashID),
	)
}

func formatGenSystemDiff(a, b gllbin.GenSystem) string {
	// Summarize gen-system field matches
	return fmt.Sprintf("label=%t key=%t type=%t company=%t info=%t",
		a.Label == b.Label,
		a.Key == b.Key,
		a.Type == b.Type,
		a.Company == b.Company,
		a.InfoText == b.InfoText,
	)
}

func formatDatabaseDiff(a, b *gllbin.Database) string {
	// Summarize database counts
	if a == nil && b == nil {
		return "nil"
	}

	return fmt.Sprintf("datafiles=%d/%d boxtypes=%d/%d sources=%d/%d filters=%d/%d",
		countDataFiles(a), countDataFiles(b),
		countBoxTypes(a), countBoxTypes(b),
		countSourceDefs(a), countSourceDefs(b),
		countFilterGroups(a), countFilterGroups(b),
	)
}

func formatResourceDiff(a, b []gllbin.Resource) string {
	// Summarize resource counts
	return fmt.Sprintf("count=%d/%d", len(a), len(b))
}

func countDataFiles(db *gllbin.Database) int {
	// Count data files safely
	if db == nil {
		return 0
	}
	return len(db.DataFiles)
}

func countBoxTypes(db *gllbin.Database) int {
	// Count box types safely
	if db == nil {
		return 0
	}
	return len(db.BoxTypes)
}

func countSourceDefs(db *gllbin.Database) int {
	// Count source definitions safely
	if db == nil {
		return 0
	}
	return len(db.SourceDefinitions)
}

func countFilterGroups(db *gllbin.Database) int {
	// Count filter groups safely
	if db == nil {
		return 0
	}
	return len(db.FilterGroups)
}
