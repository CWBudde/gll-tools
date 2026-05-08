package xgll

import (
	"errors"
	"strings"
	"testing"
)

// failingWriter returns an error after `failAfter` successful Write calls.
// Used to drive each error branch in writer_xgllbin / writer_xgllbin_pretty.
type failingWriter struct {
	calls     int
	failAfter int
}

func (f *failingWriter) Write(p []byte) (int, error) {
	f.calls++
	if f.calls > f.failAfter {
		return 0, errors.New("write fail")
	}
	return len(p), nil
}

// minimalDoc is the cheapest Document that survives JSON marshal.
func minimalDoc() *Document {
	return &Document{Statements: []Statement{{Keyword: "GLL"}}}
}

func TestXGLLBinWriter_NilDocument(t *testing.T) {
	w := xgllbinWriter{}
	err := w.Write(nil, &failingWriter{failAfter: 100})
	if err == nil || !strings.Contains(err.Error(), "nil") {
		t.Errorf("err = %v, want 'nil document' error", err)
	}
}

func TestXGLLBinWriter_WriteFailures(t *testing.T) {
	tests := []struct {
		name      string
		failAfter int
		wantSub   string
	}{
		{"magic write fails", 0, "write magic"},
		{"version write fails", 1, "write version"},
		{"length write fails", 2, "write length"},
		{"payload write fails", 3, "write payload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := xgllbinWriter{}
			err := w.Write(minimalDoc(), &failingWriter{failAfter: tt.failAfter})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("err = %q, want substring %q", err.Error(), tt.wantSub)
			}
		})
	}
}

func TestXGLLBinWriter_FormatString(t *testing.T) {
	if got := (xgllbinWriter{}).Format(); got != "xgllbin" {
		t.Errorf("Format() = %q, want xgllbin", got)
	}
}

func TestXGLLBinPrettyWriter_NilDocument(t *testing.T) {
	w := xgllbinPrettyWriter{}
	err := w.Write(nil, &failingWriter{failAfter: 100})
	if err == nil || !strings.Contains(err.Error(), "nil") {
		t.Errorf("err = %v, want 'nil document' error", err)
	}
}

func TestXGLLBinPrettyWriter_WriteFailures(t *testing.T) {
	tests := []struct {
		name      string
		failAfter int
		wantSub   string
	}{
		{"magic write fails", 0, "write magic"},
		{"version write fails", 1, "write version"},
		{"length write fails", 2, "write length"},
		{"payload write fails", 3, "write payload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := xgllbinPrettyWriter{}
			err := w.Write(minimalDoc(), &failingWriter{failAfter: tt.failAfter})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("err = %q, want substring %q", err.Error(), tt.wantSub)
			}
		})
	}
}

func TestXGLLBinPrettyWriter_FormatString(t *testing.T) {
	if got := (xgllbinPrettyWriter{}).Format(); got != "xgllbin-pretty" {
		t.Errorf("Format() = %q, want xgllbin-pretty", got)
	}
}
