package viz

import (
	"errors"
	"math"

	"github.com/cwbudde/gll-tools/internal/mesh"
	"github.com/cwbudde/gll-tools/pkg/gll"
)

func BuildBalloonMesh(def *gll.SourceDefinition, freqIndex int, dbRange, scale float64, normalize bool) (*mesh.Mesh, error) {
	if def == nil || def.BalloonData == nil {
		return nil, errors.New("no balloon data")
	}
	bd := def.BalloonData
	if len(bd.Responses) == 0 {
		return nil, errors.New("balloon responses not loaded")
	}
	ang := bd.AngularResolution
	if ang.MeridianStep <= 0 || ang.ParallelStep <= 0 {
		return nil, errors.New("invalid angular resolution")
	}

	levels, meridianCount, parallelCount, maxLevel := calculateBalloonLevels(bd, freqIndex)
	if !isFinite(maxLevel) {
		return nil, errors.New("no level data found for frequency")
	}

	displayMax := maxLevel
	if !normalize {
		displayMax = calculateGlobalMax(bd, maxLevel)
	}

	if dbRange <= 0 {
		dbRange = 40
	}
	if scale <= 0 {
		scale = 1
	}

	vertices, colors := generateBalloonVertices(levels, meridianCount, parallelCount, ang, displayMax, dbRange, scale)
	indices := generateBalloonIndices(meridianCount, parallelCount)

	return &mesh.Mesh{
		Vertices: vertices,
		Colors:   colors,
		Indices:  indices,
	}, nil
}

func calculateBalloonLevels(bd *gll.BalloonData, freqIndex int) (levels []float64, meridianCount, parallelCount int, maxLevel float64) {
	ang := bd.AngularResolution
	meridianCount = int(math.Round(360.0 / ang.MeridianStep))
	if meridianCount < 3 {
		meridianCount = 3
	}
	parallelCount = int(math.Round(180.0/ang.ParallelStep)) + 1
	if parallelCount < 2 {
		parallelCount = 2
	}

	levels = make([]float64, 0, meridianCount*parallelCount)
	maxLevel = math.NaN()

	for p := 0; p < parallelCount; p++ {
		parallelDeg := float64(p) * ang.ParallelStep
		for m := 0; m < meridianCount; m++ {
			azimuthDeg := float64(m) * ang.MeridianStep
			resp := ResponseAtAngles(bd, azimuthDeg, parallelDeg)
			level := math.NaN()
			if resp != nil && freqIndex >= 0 && freqIndex < len(resp.Level) {
				level = resp.Level[freqIndex]
			}
			levels = append(levels, level)
			if isFinite(level) {
				if !isFinite(maxLevel) || level > maxLevel {
					maxLevel = level
				}
			}
		}
	}
	return
}

func calculateGlobalMax(bd *gll.BalloonData, currentMax float64) float64 {
	globalMax := currentMax
	for i := range bd.Responses {
		levels := bd.Responses[i].Level
		for _, v := range levels {
			if isFinite(v) && v > globalMax {
				globalMax = v
			}
		}
	}
	return globalMax
}

func generateBalloonVertices(levels []float64, meridianCount, parallelCount int, ang gll.ResolutionDescriptor, displayMax, dbRange, scale float64) (vertices, colors []mesh.Vec3) {
	baseRadius := 0.3 * scale
	amplitude := 0.9 * scale
	displayMin := displayMax - dbRange

	vertices = make([]mesh.Vec3, 0, len(levels))
	colors = make([]mesh.Vec3, 0, len(levels))
	idx := 0
	for p := 0; p < parallelCount; p++ {
		parallelDeg := float64(p) * ang.ParallelStep
		phi := parallelDeg * math.Pi / 180.0
		sinPhi := math.Sin(phi)
		cosPhi := math.Cos(phi)
		for m := 0; m < meridianCount; m++ {
			azimuthDeg := float64(m) * ang.MeridianStep
			theta := azimuthDeg * math.Pi / 180.0
			level := levels[idx]
			normalized := math.NaN()
			if isFinite(level) {
				normalized = (level - displayMin) / dbRange
				if normalized < 0 {
					normalized = 0
				}
				if normalized > 1 {
					normalized = 1
				}
			}
			radius := baseRadius
			if isFinite(normalized) {
				radius = baseRadius + amplitude*normalized
			}
			x := radius * sinPhi * math.Cos(theta)
			y := radius * sinPhi * math.Sin(theta)
			z := radius * cosPhi
			vertices = append(vertices, mesh.Vec3{X: x, Y: y, Z: z})
			colors = append(colors, levelToColor(normalized))
			idx++
		}
	}
	return
}

func generateBalloonIndices(meridianCount, parallelCount int) []int {
	indices := make([]int, 0, (parallelCount-1)*meridianCount*6)
	for p := 0; p < parallelCount-1; p++ {
		for m := 0; m < meridianCount; m++ {
			nextM := (m + 1) % meridianCount
			a := p*meridianCount + m
			b := p*meridianCount + nextM
			c := (p+1)*meridianCount + m
			d := (p+1)*meridianCount + nextM
			indices = append(indices, a, c, b)
			indices = append(indices, b, c, d)
		}
	}
	return indices
}

func levelToColor(normalized float64) mesh.Vec3 {
	if !isFinite(normalized) {
		return mesh.Vec3{X: 0.65, Y: 0.65, Z: 0.65}
	}
	hue := (1 - normalized) * 0.66
	return hslToRGB(hue, 0.75, 0.5)
}

func hslToRGB(h, s, l float64) mesh.Vec3 {
	if s == 0 {
		return mesh.Vec3{X: l, Y: l, Z: l}
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	r := hueToRGB(p, q, h+1.0/3.0)
	g := hueToRGB(p, q, h)
	b := hueToRGB(p, q, h-1.0/3.0)
	return mesh.Vec3{X: r, Y: g, Z: b}
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}
