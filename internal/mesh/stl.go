package mesh

import (
	"fmt"
	"io"
	"math"
	"strings"
)

func WriteSTL(w io.Writer, mesh *Mesh, name string) error {
	if mesh == nil || len(mesh.Vertices) == 0 || len(mesh.Indices) == 0 {
		return fmt.Errorf("empty mesh")
	}
	if name == "" {
		name = defaultBalloonName
	}
	name = sanitizeSTLName(name)
	fmt.Fprintf(w, "solid %s\n", name)
	for i := 0; i+2 < len(mesh.Indices); i += 3 {
		a := mesh.Vertices[mesh.Indices[i]]
		b := mesh.Vertices[mesh.Indices[i+1]]
		c := mesh.Vertices[mesh.Indices[i+2]]
		nx, ny, nz := triangleNormal(a, b, c)
		fmt.Fprintf(w, "  facet normal %.6f %.6f %.6f\n", nx, ny, nz)
		fmt.Fprint(w, "    outer loop\n")
		fmt.Fprintf(w, "      vertex %.6f %.6f %.6f\n", a.X, a.Y, a.Z)
		fmt.Fprintf(w, "      vertex %.6f %.6f %.6f\n", b.X, b.Y, b.Z)
		fmt.Fprintf(w, "      vertex %.6f %.6f %.6f\n", c.X, c.Y, c.Z)
		fmt.Fprint(w, "    endloop\n")
		fmt.Fprint(w, "  endfacet\n")
	}
	fmt.Fprintf(w, "endsolid %s\n", name)
	return nil
}

func triangleNormal(a, b, c Vec3) (float64, float64, float64) {
	ux := b.X - a.X
	uy := b.Y - a.Y
	uz := b.Z - a.Z
	vx := c.X - a.X
	vy := c.Y - a.Y
	vz := c.Z - a.Z
	nx := uy*vz - uz*vy
	ny := uz*vx - ux*vz
	nz := ux*vy - uy*vx
	length := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if length == 0 {
		return 0, 0, 0
	}
	return nx / length, ny / length, nz / length
}

func sanitizeSTLName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultBalloonName
	}
	return strings.Map(func(r rune) rune {
		if r < 32 || r > 126 || r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return '-'
		}
		return r
	}, name)
}
