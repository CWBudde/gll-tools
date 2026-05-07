package viz

import (
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

func TestBuildCaseGeometryMeshNil(t *testing.T) {
	if _, err := BuildCaseGeometryMesh(nil); err == nil {
		t.Error("expected error for nil geometry")
	}
}

func TestBuildCaseGeometryMeshNoVertices(t *testing.T) {
	if _, err := BuildCaseGeometryMesh(&gll.CaseGeometry{}); err == nil {
		t.Error("expected error for geometry without vertices")
	}
}

func TestBuildCaseGeometryMeshFaces(t *testing.T) {
	geom := &gll.CaseGeometry{
		Vertices: []gll.Vertex{
			{X: 0, Y: 0, Z: 0},
			{X: 1, Y: 0, Z: 0},
			{X: 0, Y: 1, Z: 0},
			{X: 1, Y: 1, Z: 0},
		},
		// Vertex indices are 1-based; quad face → triangulated.
		Faces: []gll.Face{
			{Vertices: []int32{1, 2, 4, 3}},
		},
	}
	m, err := BuildCaseGeometryMesh(geom)
	if err != nil {
		t.Fatalf("BuildCaseGeometryMesh error: %v", err)
	}
	if len(m.Vertices) != 4 {
		t.Errorf("Vertices length = %d, want 4", len(m.Vertices))
	}
	// Quad triangulated → 2 triangles → 6 indices.
	if len(m.Indices) != 6 {
		t.Errorf("Indices length = %d, want 6 (2 triangles)", len(m.Indices))
	}
}

func TestBuildCaseGeometryMeshTriangle(t *testing.T) {
	geom := &gll.CaseGeometry{
		Vertices: []gll.Vertex{
			{X: 0, Y: 0, Z: 0},
			{X: 1, Y: 0, Z: 0},
			{X: 0, Y: 1, Z: 0},
		},
		Faces: []gll.Face{
			{Vertices: []int32{1, 2, 3}},
		},
	}
	m, err := BuildCaseGeometryMesh(geom)
	if err != nil {
		t.Fatalf("BuildCaseGeometryMesh error: %v", err)
	}
	if len(m.Indices) != 3 {
		t.Errorf("Indices length = %d, want 3 (1 triangle)", len(m.Indices))
	}
}

func TestBuildCaseGeometryMeshEdges(t *testing.T) {
	geom := &gll.CaseGeometry{
		Vertices: []gll.Vertex{
			{X: 0, Y: 0, Z: 0},
			{X: 1, Y: 0, Z: 0},
			{X: 0, Y: 1, Z: 0},
		},
		Edges: []gll.Edge{
			{V1: 1, V2: 2},
			{V1: 2, V2: 3},
			{V1: 3, V2: 1},
		},
	}
	m, err := BuildCaseGeometryMesh(geom)
	if err != nil {
		t.Fatalf("BuildCaseGeometryMesh error: %v", err)
	}
	if len(m.Lines) != 6 {
		t.Errorf("Lines length = %d, want 6 (3 edges × 2 verts)", len(m.Lines))
	}
	if len(m.Indices) != 0 {
		t.Errorf("expected no triangle indices when only edges are present")
	}
}

func TestBuildCaseGeometryMeshNoFacesNoEdges(t *testing.T) {
	geom := &gll.CaseGeometry{
		Vertices: []gll.Vertex{{X: 0, Y: 0, Z: 0}},
	}
	if _, err := BuildCaseGeometryMesh(geom); err == nil {
		t.Error("expected error when neither faces nor edges are present")
	}
}

func TestBuildCaseGeometryMeshMirroredVertexIndex(t *testing.T) {
	// Negative indices indicate mirrored vertices and must be treated as positive.
	geom := &gll.CaseGeometry{
		Vertices: []gll.Vertex{
			{X: 0, Y: 0, Z: 0},
			{X: 1, Y: 0, Z: 0},
			{X: 0, Y: 1, Z: 0},
		},
		Faces: []gll.Face{
			{Vertices: []int32{-1, -2, -3}},
		},
	}
	m, err := BuildCaseGeometryMesh(geom)
	if err != nil {
		t.Fatalf("BuildCaseGeometryMesh error: %v", err)
	}
	if len(m.Indices) != 3 {
		t.Errorf("Indices length = %d, want 3", len(m.Indices))
	}
}

func TestBuildCaseGeometryMeshShortFaceSkipped(t *testing.T) {
	// A face with fewer than 3 vertices should be skipped, leading to no indices.
	geom := &gll.CaseGeometry{
		Vertices: []gll.Vertex{
			{X: 0, Y: 0, Z: 0},
			{X: 1, Y: 0, Z: 0},
		},
		Faces: []gll.Face{
			{Vertices: []int32{1, 2}}, // only 2 vertices
		},
	}
	if _, err := BuildCaseGeometryMesh(geom); err == nil {
		t.Error("expected error when no triangles can be built")
	}
}
