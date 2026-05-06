package clf

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/cwbudde/gll-tools/internal/acoustics"
	"github.com/cwbudde/gll-tools/pkg/gll"
)

// ExportOption configures optional export behavior.
type ExportOption func(*exportOptions)

type exportOptions struct {
	boxes      []gll.BoxType
	sourceKey  string
	cabinetDXF string
}

// WithBoxTypes provides box type data for weight and cabinet-system fields.
// Use together with WithSourceKey to match boxes containing the exported source.
func WithBoxTypes(boxes []gll.BoxType) ExportOption {
	return func(o *exportOptions) {
		o.boxes = boxes
	}
}

// WithCabinetDXF sets the DXF file path written to the CLF <CABINET> tag.
func WithCabinetDXF(path string) ExportOption {
	return func(o *exportOptions) {
		o.cabinetDXF = path
	}
}

// WithSourceKey sets the source definition key used to look up matching box types.
func WithSourceKey(key string) ExportOption {
	return func(o *exportOptions) {
		o.sourceKey = key
	}
}

// ExportSource writes a CLF text file for a single GLL SourceDefinition.
func ExportSource(w io.Writer, src *gll.SourceDefinition, gen *gll.GenSystem, opts ...ExportOption) error {
	if src == nil {
		return fmt.Errorf("source definition is nil")
	}
	if src.BalloonData == nil {
		return fmt.Errorf("source has no balloon data")
	}

	balloon := src.BalloonData
	if len(balloon.Responses) == 0 {
		return fmt.Errorf("balloon responses not loaded (call LoadBalloonResponses first)")
	}

	angRes := balloon.AngularResolution
	merStep := angRes.MeridianStep
	parStep := angRes.ParallelStep

	// Determine CLF type from step size.
	clfType := 2
	bandFreqs := CLF2Frequencies
	if merStep >= 10 && parStep >= 10 {
		clfType = 1
		bandFreqs = CLF1Frequencies
	}

	merCount := angRes.MeridianCount()
	parCount := angRes.ParallelCount()

	// Build 3D data array: [azimuth][polar][freq].
	polarLevels := make([][][]float64, merCount)
	for m := range merCount {
		polarLevels[m] = make([][]float64, parCount)
		for p := range parCount {
			idx := acoustics.ResponseIndex(m, p, parCount, angRes.FrontHalfOnly)
			if idx >= len(balloon.Responses) {
				polarLevels[m][p] = make([]float64, len(bandFreqs))
				continue
			}

			resp := balloon.Responses[idx]
			polarLevels[m][p] = ResampleToBands(
				resp.Definition.BandsPerOctave,
				resp.Definition.StartFreq,
				resp.Level,
				bandFreqs,
			)
		}
	}

	// Build on-axis spectrum from front pole (mer=0, par=0).
	var axialSpectrum []float64
	if len(balloon.Responses) > 0 {
		resp := balloon.Responses[0]
		axialSpectrum = ResampleToBands(
			resp.Definition.BandsPerOctave,
			resp.Definition.StartFreq,
			resp.Level,
			bandFreqs,
		)
	}

	manufacturer := src.CompanyLabel
	if manufacturer == "" && gen != nil {
		manufacturer = gen.Company
	}

	radiation := defaultRadiation
	if !angRes.FrontHalfOnly {
		radiation = "fullsphere"
	}

	params := &ExportParams{
		CLFType:             clfType,
		ModelName:           src.Label,
		Manufacturer:        manufacturer,
		Description:         src.Description,
		Weight:              0,
		MinBand:             src.NominalBandwidthFrom,
		MaxBand:             src.NominalBandwidthTo,
		MeasurementDistance: src.MeasuredDistance,
		MeasurementVoltage:  fmt.Sprintf("%.3f", src.MeasuredVoltage),
		Impedance:           src.RatedImpedance,
		Radiation:           radiation,
		Symmetry:            GLLSymmetryToCLF(angRes.Symmetry),
		Reference:           defaultReference,
		AzimuthCount:        merCount,
		PolarCount:          parCount,
		StepDeg:             math.Min(merStep, parStep),
		BandFrequencies:     bandFreqs,
		PolarLevels:         polarLevels,
		AxialSpectrum:       axialSpectrum,
	}

	if gen != nil {
		params.Website = gen.WebsiteText
	}

	// Apply options.
	var o exportOptions
	for _, opt := range opts {
		opt(&o)
	}

	// Look up box types containing this source for weight and cabinet-system.
	if len(o.boxes) > 0 && o.sourceKey != "" {
		var boxLabels []string
		for _, box := range o.boxes {
			for _, sp := range box.SourcePlacements {
				if sp.SourceDefKey == o.sourceKey {
					boxLabels = append(boxLabels, box.Label)
					if box.Weight > 0 && params.Weight == 0 {
						params.Weight = box.Weight
					}

					break
				}
			}
		}
		if len(boxLabels) > 0 {
			params.CabinetSystem = strings.Join(boxLabels, ", ")
		}
	}

	if o.cabinetDXF != "" {
		params.CabinetDXF = o.cabinetDXF
	}

	return Write(w, params)
}
