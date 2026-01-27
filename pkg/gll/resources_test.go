package gll

import (
	"bytes"
	"compress/zlib"
	"testing"
)

func TestFindResourceName(t *testing.T) {
	data := []byte("\x00path/to/image.PNG\x00rest")

	name := findResourceName(data, len(data))
	if name != "path/to/image.PNG" {
		t.Fatalf("findResourceName = %q", name)
	}
}

func TestIdentifyZlibContent(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"pdf-cmap", []byte("/CIDInit foo"), "pdf-cmap"},
		{"pdf-graphics", []byte("q\n0 0 m\r\n"), "pdf-graphics"},
		{"font-ttf", []byte{0x00, 0x01, 0x00, 0x00, 0x00}, "font-ttf"},
		{"text", []byte("hello world\n"), "text"},
		{"binary", []byte{0x00, 0xFF, 0x10, 0x80}, "binary"},
	}

	for _, tc := range cases {
		if got := identifyZlibContent(tc.data); got != tc.want {
			t.Fatalf("identifyZlibContent(%s) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestScanResourcesFindsPNGAndZlib(t *testing.T) {
	pngSig := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	png := append(append(append([]byte{}, pngSig...), []byte("IEND")...), []byte{0x00, 0x00, 0x00, 0x00}...)

	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	_, _ = zw.Write([]byte("hello world hello world"))
	_ = zw.Close()

	data := []byte("xx\x00path/to/image.PNG\x00")
	data = append(data, png...)
	data = append(data, []byte("ZZ")...)
	data = append(data, compressed.Bytes()...)

	file := &File{}
	if err := scanResources(bytes.NewReader(data), file); err != nil {
		t.Fatalf("scanResources error: %v", err)
	}

	var pngFound, zlibFound bool
	for i := range file.Resources {
		res := file.Resources[i]
		switch res.Type {
		case ResourceTypePNG:
			pngFound = true
			if res.Name != "path/to/image.PNG" {
				t.Fatalf("PNG name = %q, want %q", res.Name, "path/to/image.PNG")
			}
		case ResourceTypeZlib:
			zlibFound = true
			if res.Name != "text" {
				t.Fatalf("zlib name = %q, want %q", res.Name, "text")
			}
		}
	}

	if !pngFound || !zlibFound {
		t.Fatalf("expected PNG and ZLIB resources, got %+v", file.Resources)
	}
}
