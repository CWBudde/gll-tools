package mesh

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestColorFromInt32(t *testing.T) {
	tests := []struct {
		name  string
		input int32
		wantR float64
		wantG float64
		wantB float64
	}{
		{"black", 0x000000, 0.0, 0.0, 0.0},
		{"white", 0xFFFFFF, 1.0, 1.0, 1.0},
		{"red", 0xFF0000, 1.0, 0.0, 0.0},
		{"green", 0x00FF00, 0.0, 1.0, 0.0},
		{"blue", 0x0000FF, 0.0, 0.0, 1.0},
		{"arbitrary", 0x804020, 128.0 / 255, 64.0 / 255, 32.0 / 255},
		{"zero", 0, 0.0, 0.0, 0.0},
		{"negative returns gray", -1, 0.65, 0.65, 0.65},
		{"negative large", -999999, 0.65, 0.65, 0.65},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ColorFromInt32(tt.input)
			if math.Abs(c.X-tt.wantR) > 1e-9 {
				t.Errorf("R: got %f, want %f", c.X, tt.wantR)
			}
			if math.Abs(c.Y-tt.wantG) > 1e-9 {
				t.Errorf("G: got %f, want %f", c.Y, tt.wantG)
			}
			if math.Abs(c.Z-tt.wantB) > 1e-9 {
				t.Errorf("B: got %f, want %f", c.Z, tt.wantB)
			}
		})
	}
}

func makeTriangleMesh(v0, v1, v2 Vec3) *Mesh {
	return &Mesh{
		Vertices: []Vec3{v0, v1, v2},
		Indices:  []int{0, 1, 2},
	}
}

func TestCenterMesh(t *testing.T) {
	// Triangle at (2,4,6), (4,4,6), (2,6,6)
	// BBox center: (3, 5, 6), Centroid: (8/3, 14/3, 6)
	newMesh := func() *Mesh {
		return makeTriangleMesh(
			Vec3{2, 4, 6},
			Vec3{4, 4, 6},
			Vec3{2, 6, 6},
		)
	}

	t.Run("Origin does nothing", func(t *testing.T) {
		m := newMesh()
		CenterMesh(m, CenterOrigin)
		if m.Vertices[0].X != 2 || m.Vertices[0].Y != 4 {
			t.Error("Origin mode should not modify vertices")
		}
	})

	t.Run("BBox centers to bbox midpoint", func(t *testing.T) {
		m := newMesh()
		CenterMesh(m, CenterBBox)
		// offset is (3, 5, 6), so v0 becomes (-1, -1, 0)
		assertVec3Near(t, m.Vertices[0], Vec3{-1, -1, 0})
		assertVec3Near(t, m.Vertices[1], Vec3{1, -1, 0})
		assertVec3Near(t, m.Vertices[2], Vec3{-1, 1, 0})
	})

	t.Run("Centroid centers to mean", func(t *testing.T) {
		m := newMesh()
		CenterMesh(m, CenterCentroid)
		cx := (2 + 4 + 2) / 3.0
		cy := (4 + 4 + 6) / 3.0
		assertVec3Near(t, m.Vertices[0], Vec3{2 - cx, 4 - cy, 0})
		assertVec3Near(t, m.Vertices[1], Vec3{4 - cx, 4 - cy, 0})
		assertVec3Near(t, m.Vertices[2], Vec3{2 - cx, 6 - cy, 0})
	})

	t.Run("nil mesh does not panic", func(_ *testing.T) {
		CenterMesh(nil, CenterBBox)
	})

	t.Run("empty vertices does not panic", func(_ *testing.T) {
		CenterMesh(&Mesh{}, CenterBBox)
	})
}

func TestWriteSTL(t *testing.T) {
	t.Run("single triangle", func(t *testing.T) {
		m := makeTriangleMesh(Vec3{0, 0, 0}, Vec3{1, 0, 0}, Vec3{0, 1, 0})
		var buf bytes.Buffer
		err := WriteSTL(&buf, m, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		for _, kw := range []string{"solid test", "facet normal", "outer loop", "vertex", "endloop", "endfacet", "endsolid test"} {
			if !strings.Contains(out, kw) {
				t.Errorf("output missing keyword %q", kw)
			}
		}
	})

	t.Run("empty name defaults to balloon", func(t *testing.T) {
		m := makeTriangleMesh(Vec3{0, 0, 0}, Vec3{1, 0, 0}, Vec3{0, 1, 0})
		var buf bytes.Buffer
		_ = WriteSTL(&buf, m, "")
		if !strings.Contains(buf.String(), "solid balloon") {
			t.Error("expected default name 'balloon'")
		}
	})

	t.Run("nil mesh returns error", func(t *testing.T) {
		var buf bytes.Buffer
		if err := WriteSTL(&buf, nil, "x"); err == nil {
			t.Error("expected error for nil mesh")
		}
	})

	t.Run("empty mesh returns error", func(t *testing.T) {
		var buf bytes.Buffer
		if err := WriteSTL(&buf, &Mesh{}, "x"); err == nil {
			t.Error("expected error for empty mesh")
		}
	})
}

func TestWriteOBJ(t *testing.T) {
	t.Run("single triangle", func(t *testing.T) {
		m := makeTriangleMesh(Vec3{0, 0, 0}, Vec3{1, 0, 0}, Vec3{0, 1, 0})
		var buf bytes.Buffer
		err := WriteOBJ(&buf, m, "test", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "o test") {
			t.Error("missing object name")
		}
		if !strings.Contains(out, "v ") {
			t.Error("missing vertex lines")
		}
		// OBJ faces are 1-indexed
		if !strings.Contains(out, "f 1 2 3") {
			t.Error("missing or wrong face line")
		}
	})

	t.Run("with colors", func(t *testing.T) {
		m := makeTriangleMesh(Vec3{0, 0, 0}, Vec3{1, 0, 0}, Vec3{0, 1, 0})
		m.Colors = []Vec3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
		var buf bytes.Buffer
		_ = WriteOBJ(&buf, m, "c", "")
		out := buf.String()
		// vertex lines should have 6 floats when colors present
		lines := strings.Split(out, "\n")
		for _, l := range lines {
			if strings.HasPrefix(l, "v ") {
				parts := strings.Fields(l)
				if len(parts) != 7 { // "v" + 3 coords + 3 colors
					t.Errorf("expected 7 fields in vertex line with color, got %d: %s", len(parts), l)
				}
				break
			}
		}
	})

	t.Run("with mtl reference", func(t *testing.T) {
		m := makeTriangleMesh(Vec3{0, 0, 0}, Vec3{1, 0, 0}, Vec3{0, 1, 0})
		var buf bytes.Buffer
		_ = WriteOBJ(&buf, m, "test", "material.mtl")
		out := buf.String()
		if !strings.Contains(out, "mtllib material.mtl") {
			t.Error("missing mtllib line")
		}
		if !strings.Contains(out, "usemtl test") {
			t.Error("missing usemtl line")
		}
	})

	t.Run("nil mesh returns error", func(t *testing.T) {
		var buf bytes.Buffer
		if err := WriteOBJ(&buf, nil, "x", ""); err == nil {
			t.Error("expected error for nil mesh")
		}
	})

	t.Run("lines only mesh", func(t *testing.T) {
		m := &Mesh{
			Vertices: []Vec3{{0, 0, 0}, {1, 1, 1}},
			Lines:    []int{0, 1},
		}
		var buf bytes.Buffer
		err := WriteOBJ(&buf, m, "lines", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "l 1 2") {
			t.Error("missing line element")
		}
	})
}

func TestSanitizeSTLName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"hello world", "hello-world"},
		{"", "balloon"},
		{"  ", "balloon"},
		{"tab\there", "tab-here"},
		{"line\nbreak", "line-break"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeSTLName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeSTLName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeOBJName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"hello world", "hello-world"},
		{"", "balloon"},
		{"  ", "balloon"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeOBJName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeOBJName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeMTLFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"balloon", "balloon.mtl"},
		{"", "balloon.mtl"},
		{"  ", "balloon.mtl"},
		{"my file", "my-file.mtl"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeMTLFilename(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeMTLFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWriteMTL(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMTL(&buf, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "newmtl test") {
		t.Error("missing newmtl line")
	}
	if !strings.Contains(out, "Kd") {
		t.Error("missing Kd line")
	}
}

func assertVec3Near(t *testing.T, got, want Vec3) {
	t.Helper()
	const eps = 1e-9
	if math.Abs(got.X-want.X) > eps || math.Abs(got.Y-want.Y) > eps || math.Abs(got.Z-want.Z) > eps {
		t.Errorf("got (%f, %f, %f), want (%f, %f, %f)", got.X, got.Y, got.Z, want.X, want.Y, want.Z)
	}
}
