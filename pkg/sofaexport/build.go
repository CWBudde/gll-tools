package sofaexport

import (
	"fmt"
	"strings"
	"time"

	"github.com/cwbudde/gll-tools/internal/acoustics"
	"github.com/cwbudde/gll-tools/pkg/gll"
	sofa "github.com/cwbudde/go-sofa"
)

// defaultUseCase is the placeholder label for a balloon's use case when the
// GLL provides none; also affects filename rendering and Title formatting.
const defaultUseCase = "default"

// BuildContext supplies optional GenSystem-level metadata that is otherwise
// unavailable from a SourceDefinition alone (manufacturer, etc.).
type BuildContext struct {
	Manufacturer string
	Model        string
	UseCase      string // descriptive label for the BalloonData (often defaultUseCase)
}

// BuildSOFAFile constructs an in-memory *sofa.File for one (source, balloon)
// pair. The result is suitable for passing directly to (*sofa.File).Save().
//
// The function is pure: no IO, no clock, no environment beyond the fixed
// "now" timestamp it stamps into DateCreated/DateModified.
func BuildSOFAFile(src *gll.SourceDefinition, balloon *gll.BalloonData, ctx BuildContext, opts Options) (*sofa.File, error) {
	opts = opts.withDefaults()

	if src == nil {
		return nil, fmt.Errorf("nil SourceDefinition")
	}
	if balloon == nil {
		return nil, fmt.Errorf("nil BalloonData")
	}
	if len(balloon.Responses) == 0 {
		return nil, fmt.Errorf("balloon has no responses (call gll.LoadBalloonResponses first?)")
	}

	merCount := acoustics.MeridianCount(balloon.AngularResolution.MeridianStep, int(balloon.AngularResolution.Symmetry))
	parCount := acoustics.ParallelCount(balloon.AngularResolution.ParallelStep, balloon.AngularResolution.FrontHalfOnly)
	if merCount <= 0 || parCount <= 0 {
		return nil, fmt.Errorf("invalid angular resolution: meridian=%d parallel=%d", merCount, parCount)
	}

	// All responses share the same frequency definition; take it from the first.
	first := &balloon.Responses[0]
	def := first.Definition
	n := int(def.PointCount)
	if n != len(first.Level) {
		return nil, fmt.Errorf("first response declares %d frequency points but stores %d", n, len(first.Level))
	}

	freqs := make([]float64, n)
	for i := range n {
		freqs[i] = def.GetFrequency(i)
	}

	radius := src.MeasuredDistance
	if radius <= 0 {
		radius = 1.0
	}

	m := merCount * parCount
	tfReal := make([][][]float64, m)
	tfImag := make([][][]float64, m)
	sourcePos := make([]sofa.Vector3, m)

	merStep := balloon.AngularResolution.MeridianStep
	parStep := balloon.AngularResolution.ParallelStep
	maxResp := len(balloon.Responses) - 1

	for merIdx := 0; merIdx < merCount; merIdx++ {
		for parIdx := 0; parIdx < parCount; parIdx++ {
			outRow := merIdx*parCount + parIdx

			respIdx := acoustics.ResponseIndex(merIdx, parIdx, parCount, balloon.AngularResolution.FrontHalfOnly)
			if respIdx > maxResp {
				respIdx = maxResp
			}
			resp := &balloon.Responses[respIdx]

			if int(resp.Definition.PointCount) != n {
				return nil, fmt.Errorf("response %d has PointCount=%d, want %d (mixed grids unsupported)",
					respIdx, resp.Definition.PointCount, n)
			}

			reArr, imArr, err := combineResponse(resp, src, opts.Relative)
			if err != nil {
				return nil, fmt.Errorf("combine response[%d] at (mer=%d,par=%d): %w", respIdx, merIdx, parIdx, err)
			}
			tfReal[outRow] = [][]float64{reArr}
			tfImag[outRow] = [][]float64{imArr}

			azDeg, elDeg := gridAngles(merIdx, parIdx, merStep, parStep)
			sourcePos[outRow] = directionToCartesian(azDeg, elDeg, radius)
		}
	}

	useCase := ctx.UseCase
	if useCase == "" {
		useCase = defaultUseCase
	}

	now := time.Now().UTC().Format(time.RFC3339)
	title := buildTitle(ctx, src, useCase)

	f := &sofa.File{
		Conventions:            "SOFA",
		Version:                "2.0",
		SOFAConventions:        "FreeFieldDirectivityTF",
		SOFAConventionsVersion: "1.0",
		DataType:               "TF",
		M:                      m,
		R:                      1,
		E:                      1,
		N:                      n,
		Frequencies:            freqs,
		TFReal:                 tfReal,
		TFImag:                 tfImag,
		SourcePositions:        sourcePos,
		ListenerPositions:      []sofa.Vector3{{X: 0, Y: 0, Z: 0}},
		ReceiverPositions:      []sofa.Vector3{{X: 0, Y: 0, Z: 0}},
		EmitterPositions:       []sofa.Vector3{{X: 0, Y: 0, Z: 0}},
		ListenerUp:             sofa.Vector3{X: 0, Y: 0, Z: 1},
		ListenerView:           sofa.Vector3{X: 1, Y: 0, Z: 0},
		Title:                  title,
		Organization:           ctx.Manufacturer,
		ApplicationName:        "gll-tools/sofaexport",
		ApplicationVersion:     "0.1",
		DateCreated:            now,
		DateModified:           now,
		Comment:                buildComment(src, opts.Relative),
	}
	return f, nil
}

func buildTitle(ctx BuildContext, src *gll.SourceDefinition, useCase string) string {
	parts := make([]string, 0, 4)
	if ctx.Manufacturer != "" {
		parts = append(parts, ctx.Manufacturer)
	}
	if ctx.Model != "" {
		parts = append(parts, ctx.Model)
	}
	if src.Label != "" {
		parts = append(parts, src.Label)
	}
	prefix := strings.Join(parts, " ")
	if useCase != "" && useCase != defaultUseCase {
		if prefix == "" {
			return fmt.Sprintf("[%s]", useCase)
		}
		return fmt.Sprintf("%s [%s]", prefix, useCase)
	}
	return prefix
}

func buildComment(src *gll.SourceDefinition, relative bool) string {
	mode := "combined (balloon × OnAxisSpectrum)"
	if relative {
		mode = "relative balloon (no on-axis combine)"
	}
	return fmt.Sprintf(
		"GLL→SOFA export, mode=%s. NominalBandwidth=%g..%g Hz, MeasuredVoltage=%gV, MeasuredDistance=%gm, Temperature=%g°C, Humidity=%g%%, AtmosphericPressure=%gkPa.",
		mode,
		src.NominalBandwidthFrom, src.NominalBandwidthTo,
		src.MeasuredVoltage, src.MeasuredDistance,
		src.Temperature, src.Humidity, src.AtmosphericPressure,
	)
}
