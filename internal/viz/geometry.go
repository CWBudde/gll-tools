package viz

import (
	"fmt"

	"github.com/cwbudde/gll-tools/internal/mesh"
	"github.com/cwbudde/gll-tools/pkg/gll"
)

func BuildCaseGeometryMesh(geom *gll.CaseGeometry) (*mesh.Mesh, error) {
	if geom == nil || len(geom.Vertices) == 0 {
		return nil, fmt.Errorf("no geometry vertices")
	}
	hasFaces := len(geom.Faces) > 0

	vertices := make([]mesh.Vec3, 0, len(geom.Vertices))
	colors := make([]mesh.Vec3, 0, len(geom.Vertices))
	for _, v := range geom.Vertices {
		vertices = append(vertices, mesh.Vec3{X: v.X, Y: v.Y, Z: v.Z})
		colors = append(colors, mesh.ColorFromInt32(v.Color))
	}

	indices := make([]int, 0)
	lines := make([]int, 0)

	switch {
	case hasFaces:
		for _, face := range geom.Faces {
			if len(face.Vertices) < 3 {
				continue
			}
			verts := make([]int, 0, len(face.Vertices))
			for _, idx := range face.Vertices {
				if idx == 0 {
					continue
				}
				if idx < 0 {
					idx = -idx
				}
				v := int(idx) - 1
				if v < 0 || v >= len(vertices) {
					continue
				}
				verts = append(verts, v)
			}
			if len(verts) < 3 {
				continue
			}
			for i := 1; i+1 < len(verts); i++ {
				indices = append(indices, verts[0], verts[i], verts[i+1])
			}
		}
		if len(indices) == 0 {
			return nil, fmt.Errorf("no triangle indices built from faces")
		}
	case len(geom.Edges) > 0:
		for _, edge := range geom.Edges {
			v1 := edge.V1
			v2 := edge.V2
			if v1 == 0 || v2 == 0 {
				continue
			}
			if v1 < 0 {
				v1 = -v1
			}
			if v2 < 0 {
				v2 = -v2
			}
			a := int(v1) - 1
			b := int(v2) - 1
			if a < 0 || a >= len(vertices) || b < 0 || b >= len(vertices) {
				continue
			}
			lines = append(lines, a, b)
		}
		if len(lines) == 0 {
			return nil, fmt.Errorf("no edges available to export")
		}
	default:
		return nil, fmt.Errorf("no face or edge data available")
	}

	return &mesh.Mesh{
		Vertices: vertices,
		Colors:   colors,
		Indices:  indices,
		Lines:    lines,
	}, nil
}
