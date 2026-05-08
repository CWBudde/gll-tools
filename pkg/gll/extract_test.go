package gll

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractResource(t *testing.T) {
	prefix := []byte("prefix-")
	payload := []byte("DATA")
	buf := append(append([]byte{}, prefix...), payload...)

	res := Resource{Offset: int64(len(prefix)), Size: int64(len(payload))}

	got, err := ExtractResource(bytes.NewReader(buf), res)
	if err != nil {
		t.Fatalf("ExtractResource error: %v", err)
	}

	if !bytes.Equal(got, payload) {
		t.Fatalf("ExtractResource mismatch: got %q", got)
	}
}

func TestDecompressResourceZlib(t *testing.T) {
	var compressed bytes.Buffer

	zw := zlib.NewWriter(&compressed)
	_, _ = zw.Write([]byte("hello"))
	_ = zw.Close()

	prefix := []byte("xx")
	buf := append(append([]byte{}, prefix...), compressed.Bytes()...)

	res := Resource{Type: ResourceTypeZlib, Offset: int64(len(prefix)), Size: int64(compressed.Len())}

	got, err := DecompressResource(bytes.NewReader(buf), res)
	if err != nil {
		t.Fatalf("DecompressResource error: %v", err)
	}

	if string(got) != "hello" {
		t.Fatalf("DecompressResource mismatch: got %q", string(got))
	}
}

func TestDecompressResourceNonZlib(t *testing.T) {
	buf := []byte("abcdef")
	res := Resource{Type: ResourceTypePNG, Offset: 2, Size: 3}

	got, err := DecompressResource(bytes.NewReader(buf), res)
	if err != nil {
		t.Fatalf("DecompressResource error: %v", err)
	}

	if string(got) != "cde" {
		t.Fatalf("DecompressResource mismatch: got %q", string(got))
	}
}

func TestCalculateChecksum(t *testing.T) {
	data := []byte{1, 2, 3, 4}

	want := [4]byte{232, 188, 60, 215}
	if got := CalculateChecksum(data, 0, len(data)); got != want {
		t.Fatalf("CalculateChecksum = %v, want %v", got, want)
	}
}

// ---- ExtractDataFile ----

func TestExtractDataFile(t *testing.T) {
	prefix := []byte("hdr-stuff-")
	payload := []byte("EMBEDDED-XED-CONTENT")
	buf := append(append([]byte{}, prefix...), payload...)

	df := DataFile{
		Key:      "k1",
		Filename: "geometry.xed",
		Size:     int32(len(payload)), //nolint:gosec // payload is a small literal in tests
		Offset:   int64(len(prefix)),
	}

	got, err := ExtractDataFile(bytes.NewReader(buf), df)
	if err != nil {
		t.Fatalf("ExtractDataFile error: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("ExtractDataFile mismatch: got %q, want %q", got, payload)
	}
}

func TestExtractDataFile_ZeroSize(t *testing.T) {
	df := DataFile{Size: 0, Offset: 4}
	got, err := ExtractDataFile(bytes.NewReader([]byte("0123456789")), df)
	if err != nil {
		t.Fatalf("ExtractDataFile zero-size error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ExtractDataFile zero-size = %q, want empty", got)
	}
}

func TestExtractDataFile_SeekError(t *testing.T) {
	df := DataFile{Size: 4, Offset: -1}
	_, err := ExtractDataFile(bytes.NewReader([]byte("abcdefgh")), df)
	if err == nil {
		t.Fatal("ExtractDataFile with negative offset: expected error, got nil")
	}
}

func TestExtractDataFile_ShortReader(t *testing.T) {
	// Size claims more bytes than the reader has after Offset.
	df := DataFile{Size: 100, Offset: 2}
	_, err := ExtractDataFile(bytes.NewReader([]byte("abcdef")), df)
	if err == nil {
		t.Fatal("ExtractDataFile with truncated reader: expected error, got nil")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		t.Errorf("ExtractDataFile short read err = %v, want EOF/UnexpectedEOF", err)
	}
}

// ---- ExtractIncludeFile ----

func TestExtractIncludeFile(t *testing.T) {
	prefix := bytes.Repeat([]byte{0xCC}, 17)
	payload := []byte("%PDF-1.4 fake-spec-sheet bytes")
	buf := append(append([]byte{}, prefix...), payload...)

	inc := IncludeFile{
		Key:      "doc1",
		Label:    "Spec sheet",
		Filename: "spec.pdf",
		Size:     int32(len(payload)), //nolint:gosec // payload is a small literal in tests
		Offset:   int64(len(prefix)),
	}

	got, err := ExtractIncludeFile(bytes.NewReader(buf), inc)
	if err != nil {
		t.Fatalf("ExtractIncludeFile error: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("ExtractIncludeFile mismatch: got %q, want %q", got, payload)
	}
}

func TestExtractIncludeFile_ZeroSize(t *testing.T) {
	inc := IncludeFile{Size: 0, Offset: 0}
	got, err := ExtractIncludeFile(bytes.NewReader(nil), inc)
	if err != nil {
		t.Fatalf("ExtractIncludeFile zero-size error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ExtractIncludeFile zero-size = %q, want empty", got)
	}
}

func TestExtractIncludeFile_SeekError(t *testing.T) {
	inc := IncludeFile{Size: 4, Offset: -10}
	_, err := ExtractIncludeFile(bytes.NewReader([]byte("xxxx")), inc)
	if err == nil {
		t.Fatal("ExtractIncludeFile with negative offset: expected error, got nil")
	}
}

func TestExtractIncludeFile_ShortReader(t *testing.T) {
	inc := IncludeFile{Size: 50, Offset: 0}
	_, err := ExtractIncludeFile(bytes.NewReader([]byte("only-12-bytes")), inc)
	if err == nil {
		t.Fatal("ExtractIncludeFile with truncated reader: expected error, got nil")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		t.Errorf("ExtractIncludeFile short read err = %v, want EOF/UnexpectedEOF", err)
	}
}

// ---- Integration: extract every embedded file from sample GLLs ----

// TestExtractDataFile_FromRealFiles parses each sample, then extracts every
// DataFile and verifies the returned byte slice matches the recorded size.
// This exercises the same code path used by the gllinfo extract subcommand.
func TestExtractDataFile_FromRealFiles(t *testing.T) {
	files := []string{
		"D12-v10.gll",
		"D20-V10.gll",
		"TiRAY-V1_3.gll",
		"APS-V1_1.gll",
		"Coda-Audio G-Series-V1_2.gll",
		"LX-10 ASX_gll.gll",
		"LX-20 ASX_gll.gll",
		"LX-60 ASX_gll.gll",
		"HOPS7-Pro V1_0.gll",
		"D12 reduced.gll",
		"N-RAY-V0_3 Beta.gll",
		"N-APS v1_0.gll",
	}

	totalExtracted := 0
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", "gll", name)
			f, err := os.Open(path)
			if err != nil {
				t.Skipf("test file not found: %v", err)
			}
			t.Cleanup(func() { f.Close() })

			gllFile, err := Parse(f)
			if err != nil {
				t.Fatalf("Parse(%q): %v", name, err)
			}
			if gllFile.Database == nil || len(gllFile.Database.DataFiles) == 0 {
				t.Skip("no data files")
			}

			extracted := 0
			for i, df := range gllFile.Database.DataFiles {
				data, err := ExtractDataFile(f, df)
				if err != nil {
					t.Errorf("DataFiles[%d] (%q): extract error: %v", i, df.Filename, err)
					continue
				}
				if int64(len(data)) != int64(df.Size) { //nolint:gosec // size is non-negative by construction
					t.Errorf("DataFiles[%d] (%q): got %d bytes, want %d",
						i, df.Filename, len(data), df.Size)
				}
				extracted++
			}
			totalExtracted += extracted
			t.Logf("extracted %d data files from %s", extracted, name)
		})
	}

	if totalExtracted == 0 {
		t.Skip("no real data files available; synthetic tests still validate the extractor")
	}
}

// TestExtractIncludeFile_FromRealFiles is the IncludeFile counterpart.
func TestExtractIncludeFile_FromRealFiles(t *testing.T) {
	files := []string{
		"D12-v10.gll",
		"D20-V10.gll",
		"TiRAY-V1_3.gll",
		"APS-V1_1.gll",
		"Coda-Audio G-Series-V1_2.gll",
		"LX-10 ASX_gll.gll",
		"LX-20 ASX_gll.gll",
		"LX-60 ASX_gll.gll",
		"HOPS7-Pro V1_0.gll",
		"D12 reduced.gll",
		"N-RAY-V0_3 Beta.gll",
		"N-APS v1_0.gll",
	}

	totalExtracted := 0
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", "gll", name)
			f, err := os.Open(path)
			if err != nil {
				t.Skipf("test file not found: %v", err)
			}
			t.Cleanup(func() { f.Close() })

			gllFile, err := Parse(f)
			if err != nil {
				t.Fatalf("Parse(%q): %v", name, err)
			}
			if gllFile.Database == nil || len(gllFile.Database.IncludeFiles) == 0 {
				t.Skip("no include files")
			}

			extracted := 0
			for i, inc := range gllFile.Database.IncludeFiles {
				data, err := ExtractIncludeFile(f, inc)
				if err != nil {
					t.Errorf("IncludeFiles[%d] (%q): extract error: %v", i, inc.Filename, err)
					continue
				}
				if int64(len(data)) != int64(inc.Size) { //nolint:gosec // size is non-negative by construction
					t.Errorf("IncludeFiles[%d] (%q): got %d bytes, want %d",
						i, inc.Filename, len(data), inc.Size)
				}
				extracted++
			}
			totalExtracted += extracted
			t.Logf("extracted %d include files from %s", extracted, name)
		})
	}

	if totalExtracted == 0 {
		t.Skip("no real include files available; synthetic tests still validate the extractor")
	}
}
