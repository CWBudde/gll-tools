package mesh

import (
	"fmt"
	"io"
	"math"
)

// WriteDXF writes the mesh as an ASCII DXF file (R12/AC1009 compatible).
// Faces are written as 3DFACE entities, edges as LINE entities.
// Colors are mapped to the nearest AutoCAD Color Index (ACI).
func WriteDXF(w io.Writer, m *Mesh, _ string) error {
	if m == nil || len(m.Vertices) == 0 || (len(m.Indices) == 0 && len(m.Lines) == 0) {
		return fmt.Errorf("empty mesh")
	}

	// HEADER section.
	fmt.Fprint(w, "0\nSECTION\n2\nHEADER\n")
	fmt.Fprint(w, "9\n$ACADVER\n1\nAC1009\n")
	fmt.Fprint(w, "0\nENDSEC\n")

	// ENTITIES section.
	fmt.Fprint(w, "0\nSECTION\n2\nENTITIES\n")

	hasColors := len(m.Colors) == len(m.Vertices)

	// Write 3DFACE entities for triangulated faces.
	for i := 0; i+2 < len(m.Indices); i += 3 {
		ai, bi, ci := m.Indices[i], m.Indices[i+1], m.Indices[i+2]
		a, b, c := m.Vertices[ai], m.Vertices[bi], m.Vertices[ci]

		color := 7 // white/default
		if hasColors {
			color = rgbToACI(m.Colors[ai])
		}

		fmt.Fprint(w, "0\n3DFACE\n")
		fmt.Fprintf(w, "8\n%s\n", "0") // layer
		fmt.Fprintf(w, "62\n%d\n", color)
		fmt.Fprintf(w, "10\n%.6f\n20\n%.6f\n30\n%.6f\n", a.X, a.Y, a.Z)
		fmt.Fprintf(w, "11\n%.6f\n21\n%.6f\n31\n%.6f\n", b.X, b.Y, b.Z)
		fmt.Fprintf(w, "12\n%.6f\n22\n%.6f\n32\n%.6f\n", c.X, c.Y, c.Z)
		// DXF 3DFACE requires 4 corners; repeat last vertex for triangles.
		fmt.Fprintf(w, "13\n%.6f\n23\n%.6f\n33\n%.6f\n", c.X, c.Y, c.Z)
	}

	// Write LINE entities for edges.
	for i := 0; i+1 < len(m.Lines); i += 2 {
		ai, bi := m.Lines[i], m.Lines[i+1]
		a, b := m.Vertices[ai], m.Vertices[bi]

		color := 7
		if hasColors {
			color = rgbToACI(m.Colors[ai])
		}

		fmt.Fprint(w, "0\nLINE\n")
		fmt.Fprintf(w, "8\n%s\n", "0")
		fmt.Fprintf(w, "62\n%d\n", color)
		fmt.Fprintf(w, "10\n%.6f\n20\n%.6f\n30\n%.6f\n", a.X, a.Y, a.Z)
		fmt.Fprintf(w, "11\n%.6f\n21\n%.6f\n31\n%.6f\n", b.X, b.Y, b.Z)
	}

	fmt.Fprint(w, "0\nENDSEC\n")

	// EOF.
	fmt.Fprint(w, "0\nEOF\n")

	return nil
}

// aciColors maps ACI index to RGB (0-255).
// Only the 9 standard colors (1-9) plus white (7) are used for mapping.
var aciColors = []struct {
	R, G, B int
	ACI     int
}{
	{255, 0, 0, 1},     // red
	{255, 255, 0, 2},   // yellow
	{0, 255, 0, 3},     // green
	{0, 255, 255, 4},   // cyan
	{0, 0, 255, 5},     // blue
	{255, 0, 255, 6},   // magenta
	{255, 255, 255, 7}, // white
	{128, 128, 128, 8}, // dark gray
	{192, 192, 192, 9}, // light gray
}

// rgbToACI maps a normalized RGB color (0-1) to the nearest ACI color index.
func rgbToACI(c Vec3) int {
	r := int(math.Round(c.X * 255))
	g := int(math.Round(c.Y * 255))
	b := int(math.Round(c.Z * 255))

	bestACI := 7
	bestDist := math.MaxFloat64

	for _, ac := range aciColors {
		dr := float64(r - ac.R)
		dg := float64(g - ac.G)
		db := float64(b - ac.B)
		dist := dr*dr + dg*dg + db*db
		if dist < bestDist {
			bestDist = dist
			bestACI = ac.ACI
		}
	}

	return bestACI
}
