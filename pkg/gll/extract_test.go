package gll

import (
	"bytes"
	"compress/zlib"
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
