package mesh

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteDXF_faces(t *testing.T) {
	m := &Mesh{
		Vertices: []Vec3{
			{0, 0, 0},
			{1, 0, 0},
			{0, 1, 0},
		},
		Colors: []Vec3{
			{1, 0, 0}, // red
			{1, 0, 0},
			{1, 0, 0},
		},
		Indices: []int{0, 1, 2},
	}

	var buf bytes.Buffer
	err := WriteDXF(&buf, m, "test")
	if err != nil {
		t.Fatalf("WriteDXF failed: %v", err)
	}

	output := buf.String()

	checks := []struct {
		desc string
		want string
	}{
		{"header", "$ACADVER"},
		{"version", "AC1009"},
		{"3dface entity", "3DFACE"},
		{"aci red", "\n62\n1\n"},
		{"eof", "EOF"},
		{"endsec", "ENDSEC"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.want) {
			t.Errorf("missing %s: want %q", c.desc, c.want)
		}
	}
}

func TestWriteDXF_lines(t *testing.T) {
	m := &Mesh{
		Vertices: []Vec3{
			{0, 0, 0},
			{1, 0, 0},
		},
		Lines: []int{0, 1},
	}

	var buf bytes.Buffer
	err := WriteDXF(&buf, m, "test")
	if err != nil {
		t.Fatalf("WriteDXF failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "LINE") {
		t.Error("missing LINE entity")
	}
	if !strings.Contains(output, "1.000000") {
		t.Error("missing vertex coordinate")
	}
}

func TestWriteDXF_empty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteDXF(&buf, &Mesh{}, "test")
	if err == nil {
		t.Error("expected error for empty mesh")
	}
}

func TestWriteDXF_nil(t *testing.T) {
	var buf bytes.Buffer
	err := WriteDXF(&buf, nil, "test")
	if err == nil {
		t.Error("expected error for nil mesh")
	}
}

func TestWriteDXF_defaultColor(t *testing.T) {
	m := &Mesh{
		Vertices: []Vec3{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
		Indices:  []int{0, 1, 2},
	}

	var buf bytes.Buffer
	err := WriteDXF(&buf, m, "test")
	if err != nil {
		t.Fatalf("WriteDXF failed: %v", err)
	}

	// Without colors, should use ACI 7 (white/default).
	if !strings.Contains(buf.String(), "\n62\n7\n") {
		t.Error("expected default ACI color 7")
	}
}

func TestRgbToACI(t *testing.T) {
	tests := []struct {
		name string
		rgb  Vec3
		want int
	}{
		{"pure red", Vec3{1, 0, 0}, 1},
		{"pure green", Vec3{0, 1, 0}, 3},
		{"pure blue", Vec3{0, 0, 1}, 5},
		{"white", Vec3{1, 1, 1}, 7},
		{"gray", Vec3{0.5, 0.5, 0.5}, 8},
		{"yellow", Vec3{1, 1, 0}, 2},
		{"cyan", Vec3{0, 1, 1}, 4},
		{"magenta", Vec3{1, 0, 1}, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rgbToACI(tt.rgb)
			if got != tt.want {
				t.Errorf("rgbToACI(%v) = %d, want %d", tt.rgb, got, tt.want)
			}
		})
	}
}
