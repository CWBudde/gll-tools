package mesh

import (
	"fmt"
	"io"
	"strings"
)

func WriteOBJ(w io.Writer, mesh *Mesh, name string, mtlName string) error {
	if mesh == nil || len(mesh.Vertices) == 0 || (len(mesh.Indices) == 0 && len(mesh.Lines) == 0) {
		return fmt.Errorf("empty mesh")
	}
	if name == "" {
		name = defaultBalloonName
	}
	name = sanitizeOBJName(name)
	if mtlName != "" {
		fmt.Fprintf(w, "mtllib %s\n", sanitizeOBJName(mtlName))
	}
	fmt.Fprintf(w, "o %s\n", name)
	hasColors := len(mesh.Colors) == len(mesh.Vertices)
	for i, v := range mesh.Vertices {
		if hasColors {
			c := mesh.Colors[i]
			fmt.Fprintf(w, "v %.6f %.6f %.6f %.6f %.6f %.6f\n", v.X, v.Y, v.Z, c.X, c.Y, c.Z)
		} else {
			fmt.Fprintf(w, "v %.6f %.6f %.6f\n", v.X, v.Y, v.Z)
		}
	}
	if mtlName != "" {
		fmt.Fprintf(w, "usemtl %s\n", sanitizeOBJName(name))
	}
	for i := 0; i+2 < len(mesh.Indices); i += 3 {
		a := mesh.Indices[i] + 1
		b := mesh.Indices[i+1] + 1
		c := mesh.Indices[i+2] + 1
		fmt.Fprintf(w, "f %d %d %d\n", a, b, c)
	}
	for i := 0; i+1 < len(mesh.Lines); i += 2 {
		a := mesh.Lines[i] + 1
		b := mesh.Lines[i+1] + 1
		fmt.Fprintf(w, "l %d %d\n", a, b)
	}
	return nil
}

func sanitizeOBJName(name string) string {
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
