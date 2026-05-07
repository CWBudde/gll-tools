package cmd

import (
	"bytes"
	"compress/zlib"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTailCommand(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "tail", path); err != nil {
		t.Fatalf("tail command failed: %v", err)
	}
}

// TestTailCommandWithTailData exercises the tail command on a file that has
// non-empty trailing data, which engages the print helpers (records, strings,
// presets) that the empty-tail file does not.
func TestTailCommandWithTailData(t *testing.T) {
	path := filepath.FromSlash(testGLLFileFilters)
	if err := runRoot(t, "tail", path); err != nil {
		t.Fatalf("tail command on filters file failed: %v", err)
	}
}

func TestTailCommandMissingFile(t *testing.T) {
	if err := runRoot(t, "tail", "nonexistent.gll"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLog2(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{1, 0},
		{2, 1},
		{4, 2},
		{8, 3},
		{0.5, -1},
	}
	for _, tc := range cases {
		got := log2(tc.in)
		if math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("log2(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestShannonEntropy(t *testing.T) {
	if e := shannonEntropy(nil); e != 0 {
		t.Errorf("shannonEntropy(nil) = %v, want 0", e)
	}
	if e := shannonEntropy([]byte{}); e != 0 {
		t.Errorf("shannonEntropy(empty) = %v, want 0", e)
	}

	uniform := bytes.Repeat([]byte{0x42}, 64)
	if e := shannonEntropy(uniform); e != 0 {
		t.Errorf("shannonEntropy(uniform) = %v, want 0", e)
	}

	twoVals := append(bytes.Repeat([]byte{0}, 8), bytes.Repeat([]byte{1}, 8)...)
	if e := shannonEntropy(twoVals); math.Abs(e-1.0) > 1e-9 {
		t.Errorf("shannonEntropy(50/50 two-symbol) = %v, want 1.0", e)
	}

	// Uniform distribution over all 256 values yields exactly 8 bits/byte.
	full := make([]byte, 256)
	for i := range full {
		full[i] = byte(i)
	}
	if e := shannonEntropy(full); math.Abs(e-8.0) > 1e-9 {
		t.Errorf("shannonEntropy(full alphabet) = %v, want 8.0", e)
	}
}

func TestCountPrintable(t *testing.T) {
	cases := []struct {
		name           string
		in             []byte
		wantPrintable  int
		wantTotal      int
	}{
		{"empty", nil, 0, 0},
		{"all printable", []byte("Hello, World!"), 13, 13},
		{"none printable", []byte{0x00, 0x01, 0x1F, 0x7F, 0xFF}, 0, 5},
		{"mixed", []byte{'A', 0x00, 'B', 0x7F, 'C'}, 3, 5},
		{"boundary 0x20 included", []byte{' '}, 1, 1},
		{"boundary 0x7E included", []byte{'~'}, 1, 1},
		{"boundary 0x1F excluded", []byte{0x1F}, 0, 1},
		{"boundary 0x7F excluded", []byte{0x7F}, 0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, total := countPrintable(tc.in)
			if p != tc.wantPrintable || total != tc.wantTotal {
				t.Errorf("countPrintable = (%d,%d), want (%d,%d)",
					p, total, tc.wantPrintable, tc.wantTotal)
			}
		})
	}
}

func TestIsZlibSignature(t *testing.T) {
	cases := []struct {
		kind string
		want bool
	}{
		{"ZLIB(78 9C)", true},
		{"ZLIB(78 5E)", true},
		{"ZLIB(78 DA)", true},
		{"PNG", false},
		{"PDF", false},
		{"ZIP", false},
		{"", false},
		{"zlib(78 9c)", false}, // case-sensitive
	}
	for _, tc := range cases {
		if got := isZlibSignature(tc.kind); got != tc.want {
			t.Errorf("isZlibSignature(%q) = %v, want %v", tc.kind, got, tc.want)
		}
	}
}

func TestScanTailSignaturesNone(t *testing.T) {
	if matches := scanTailSignatures([]byte("nothing interesting here")); len(matches) != 0 {
		t.Errorf("expected no matches, got %v", matches)
	}
	if matches := scanTailSignatures(nil); len(matches) != 0 {
		t.Errorf("expected no matches on nil, got %v", matches)
	}
}

func TestScanTailSignaturesFindsAll(t *testing.T) {
	pngSig := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	zlibSig := []byte{0x78, 0x9C}
	pdfSig := []byte("%PDF")
	zipSig := []byte{'P', 'K', 0x03, 0x04}

	var data []byte
	data = append(data, []byte("AAA")...) // padding 0..2
	data = append(data, pngSig...)        // 3..10
	data = append(data, []byte("BB")...)  // 11..12
	data = append(data, zlibSig...)       // 13..14
	data = append(data, []byte("C")...)   // 15
	data = append(data, pdfSig...)        // 16..19
	data = append(data, zipSig...)        // 20..23

	matches := scanTailSignatures(data)

	want := map[string]int{
		"PNG":         3,
		"ZLIB(78 9C)": 13,
		"PDF":         16,
		"ZIP":         20,
	}
	got := map[string]int{}
	for _, m := range matches {
		// keep first occurrence per kind
		if _, ok := got[m.kind]; !ok {
			got[m.kind] = m.offset
		}
	}
	for kind, off := range want {
		if got[kind] != off {
			t.Errorf("kind %q: got offset %d, want %d", kind, got[kind], off)
		}
	}
}

func TestScanTailSignaturesOverlapping(t *testing.T) {
	// Two PNG signatures back-to-back must both be reported.
	pngSig := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	data := append(append([]byte{}, pngSig...), pngSig...)

	matches := scanTailSignatures(data)
	pngOffsets := []int{}
	for _, m := range matches {
		if m.kind == "PNG" {
			pngOffsets = append(pngOffsets, m.offset)
		}
	}
	if len(pngOffsets) != 2 || pngOffsets[0] != 0 || pngOffsets[1] != 8 {
		t.Errorf("expected PNG matches at offsets [0, 8], got %v", pngOffsets)
	}
}

// captureStdout swaps os.Stdout for a pipe, runs fn, and returns what was printed.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}

func TestDumpHexWindow(t *testing.T) {
	data := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77}

	out := captureStdout(t, func() { dumpHexWindow(data, 4, 2) })
	// radius 2 around offset 4 → bytes [2:6] = 22 33 44 55
	if !strings.Contains(out, "22334455") {
		t.Errorf("expected hex 22334455 in output, got: %s", out)
	}

	// radius beyond bounds clamps to [0, len].
	outFull := captureStdout(t, func() { dumpHexWindow(data, 0, 100) })
	if !strings.Contains(outFull, "0011223344556677") {
		t.Errorf("expected full hex string, got: %s", outFull)
	}
}

func TestCheckZlibValid(t *testing.T) {
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write([]byte("hello world")); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}

	out := captureStdout(t, func() { checkZlib(compressed.Bytes(), 1024) })
	if !strings.Contains(out, "zlib: ok") {
		t.Errorf("expected 'zlib: ok' in output, got: %s", out)
	}
	if !strings.Contains(out, "11 bytes") {
		t.Errorf("expected '11 bytes' in output, got: %s", out)
	}
}

func TestCheckZlibInvalid(t *testing.T) {
	out := captureStdout(t, func() { checkZlib([]byte{0x78, 0x9C, 0x00, 0x00}, 1024) })
	if !strings.Contains(out, "zlib:") {
		t.Errorf("expected zlib diagnostic output, got: %s", out)
	}
	if strings.Contains(out, "ok") {
		t.Errorf("expected failure, got success: %s", out)
	}
}

func TestCheckZlibTruncatedAtCap(t *testing.T) {
	// Compress more than the cap so the truncation branch fires.
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	payload := bytes.Repeat([]byte("X"), 2048)
	if _, err := zw.Write(payload); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}

	out := captureStdout(t, func() { checkZlib(compressed.Bytes(), 16) })
	if !strings.Contains(out, "truncated") {
		t.Errorf("expected truncation message, got: %s", out)
	}
}

func TestPrintTailStats(t *testing.T) {
	out := captureStdout(t, func() { printTailStats([]byte("ABCDEFGH 12345678 ........")) })
	if !strings.Contains(out, "Entropy:") {
		t.Errorf("expected entropy line, got: %s", out)
	}
	if !strings.Contains(out, "ASCII printable:") {
		t.Errorf("expected printable line, got: %s", out)
	}
}

func TestPrintHexDump(t *testing.T) {
	data := []byte{0x41, 0x42, 0x43, 0x00, 0x7F}
	out := captureStdout(t, func() { printHexDump(data) })
	if !strings.Contains(out, "Hex dump:") {
		t.Errorf("missing header, got: %s", out)
	}
	if !strings.Contains(out, "41 42 43") {
		t.Errorf("expected hex bytes in output, got: %s", out)
	}
	// Printable subset shown alongside, non-printables become '.'.
	if !strings.Contains(out, "ABC") {
		t.Errorf("expected ASCII column to contain ABC, got: %s", out)
	}
}

func TestPrintASCIIRuns(t *testing.T) {
	// Two runs separated by a non-printable, plus a short run that's filtered out.
	data := []byte("hello\x00world\x00ab")
	out := captureStdout(t, func() { printASCIIRuns(data, 4) })
	if !strings.Contains(out, "hello") || !strings.Contains(out, "world") {
		t.Errorf("expected hello and world runs, got: %s", out)
	}
	if strings.Contains(out, `"ab"`) {
		t.Errorf("short run should have been filtered, got: %s", out)
	}

	// No runs case.
	outNone := captureStdout(t, func() { printASCIIRuns([]byte{0x00, 0x01, 0x02}, 4) })
	if !strings.Contains(outNone, "none") {
		t.Errorf("expected 'none' for no runs, got: %s", outNone)
	}
}

func TestPrintASCIIRunsTrailing(t *testing.T) {
	// Run that extends to end-of-buffer without a trailing terminator.
	data := []byte("\x00trailing")
	out := captureStdout(t, func() { printASCIIRuns(data, 4) })
	if !strings.Contains(out, "trailing") {
		t.Errorf("expected trailing run, got: %s", out)
	}
}

func TestPrintRepeats(t *testing.T) {
	pattern := bytes.Repeat([]byte("ABCDEFGHIJKLMNOP"), 3) // 48 bytes, repeats every 16
	out := captureStdout(t, func() { printRepeats(pattern, 16) })
	if !strings.Contains(out, "Repeats (window 16):") {
		t.Errorf("expected repeats header, got: %s", out)
	}
	if !strings.Contains(out, "==") {
		t.Errorf("expected at least one match line, got: %s", out)
	}
}

func TestPrintRepeatsNoneTooShort(t *testing.T) {
	out := captureStdout(t, func() { printRepeats([]byte("short"), 16) })
	if !strings.Contains(out, "none") {
		t.Errorf("expected 'none' when input shorter than 2*window, got: %s", out)
	}
}

func TestPrintRepeatsNoneNoMatches(t *testing.T) {
	// 64 unique bytes, no 16-byte window repeats.
	data := make([]byte, 64)
	for i := range data {
		data[i] = byte(i)
	}
	out := captureStdout(t, func() { printRepeats(data, 16) })
	if !strings.Contains(out, "none") {
		t.Errorf("expected 'none' for non-repeating data, got: %s", out)
	}
}
