package mesh

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

type Mesh struct {
	Vertices []Vec3
	Colors   []Vec3
	Indices  []int
	Lines    []int
}

type CenterMode string

const (
	CenterOrigin   CenterMode = "origin"
	CenterBBox     CenterMode = "bbox"
	CenterCentroid CenterMode = "centroid"
)

const defaultBalloonName = "balloon"

func CenterMesh(m *Mesh, mode CenterMode) {
	if m == nil || len(m.Vertices) == 0 || mode == CenterOrigin {
		return
	}

	var offset Vec3
	switch mode {
	case CenterBBox:
		minValue := m.Vertices[0]
		maxValue := m.Vertices[0]
		for _, v := range m.Vertices[1:] {
			if v.X < minValue.X {
				minValue.X = v.X
			}
			if v.Y < minValue.Y {
				minValue.Y = v.Y
			}
			if v.Z < minValue.Z {
				minValue.Z = v.Z
			}
			if v.X > maxValue.X {
				maxValue.X = v.X
			}
			if v.Y > maxValue.Y {
				maxValue.Y = v.Y
			}
			if v.Z > maxValue.Z {
				maxValue.Z = v.Z
			}
		}
		offset = Vec3{
			X: (minValue.X + maxValue.X) / 2,
			Y: (minValue.Y + maxValue.Y) / 2,
			Z: (minValue.Z + maxValue.Z) / 2,
		}
	case CenterCentroid:
		var sum Vec3
		for _, v := range m.Vertices {
			sum.X += v.X
			sum.Y += v.Y
			sum.Z += v.Z
		}
		n := float64(len(m.Vertices))
		offset = Vec3{X: sum.X / n, Y: sum.Y / n, Z: sum.Z / n}
	default:
		return
	}

	for i := range m.Vertices {
		m.Vertices[i].X -= offset.X
		m.Vertices[i].Y -= offset.Y
		m.Vertices[i].Z -= offset.Z
	}
}

func ColorFromInt32(value int32) Vec3 {
	if value < 0 {
		return Vec3{X: 0.65, Y: 0.65, Z: 0.65}
	}
	r := float64((value >> 16) & 0xFF)
	g := float64((value >> 8) & 0xFF)
	b := float64(value & 0xFF)
	return Vec3{X: r / 255.0, Y: g / 255.0, Z: b / 255.0}
}
