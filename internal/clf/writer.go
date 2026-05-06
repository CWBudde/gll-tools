package clf

import (
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	defaultRadiation = "halfsphere"
	defaultReference = "absolute"
)

// ExportParams contains all data needed to write a CLF text file.
type ExportParams struct {
	CLFType int // 1 or 2

	// Header metadata
	ModelName    string
	Manufacturer string
	Description  string
	Website      string
	Colors       string
	Mounting     string
	Weight       float64
	MinBand      float64
	MaxBand      float64

	// Measurement info
	MeasurementContact     string
	MeasurementEmail       string
	MeasurementDate        time.Time
	MeasurementNote        string
	MeasurementEnvironment string
	MeasurementDistance    float64
	MeasurementVoltage     string

	// Speaker type
	SpeakerType string
	Impedance   float64
	Sensitivity []float64

	// Radiation and symmetry
	Radiation string
	Symmetry  string
	Reference string

	// Angular grid
	AzimuthCount int
	PolarCount   int
	StepDeg      float64

	// Directivity data: [azimuth][polar][freq]
	BandFrequencies []float64
	PolarLevels     [][][]float64

	// Optional on-axis spectrum
	AxialSpectrum []float64

	// Optional cabinet info
	CabinetDXF    string // path to DXF file for <CABINET> tag
	CabinetSystem string
}

// Write writes a CLF text format file to the given writer.
func Write(w io.Writer, p *ExportParams) error {
	clfType := p.CLFType
	if clfType != 1 && clfType != 2 {
		clfType = 2
	}

	tag := fmt.Sprintf("CLF%d", clfType)
	endTag := fmt.Sprintf("<%sEND>", tag)

	wl := func(format string, args ...any) {
		fmt.Fprintf(w, format+"\n", args...)
	}

	date := p.MeasurementDate
	if date.IsZero() {
		date = time.Now()
	}
	months := []string{"", "JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}
	dateStr := fmt.Sprintf("%d-%s-%d", date.Year(), months[date.Month()], date.Day())

	or := func(s, fallback string) string {
		if s == "" {
			return fallback
		}
		return s
	}

	// Header
	wl("<%s>", tag)
	wl("<VERSION>\t1")
	wl("<MODELNAME>\t%s", or(p.ModelName, "#"))
	wl("<INFOFILE>\t#")
	wl("<MANUFACTURER>\t%s", or(p.Manufacturer, "#"))
	wl("<WEB-SITE>\t%s", or(p.Website, "#"))
	wl("<DESCRIPTION>\t%s", or(p.Description, "#"))
	wl("<COLORS>\t%s", or(p.Colors, "#"))
	wl("<MOUNTING>\t%s", or(p.Mounting, "#"))
	wl("<WEIGHT>\t%.1f", p.Weight)
	wl("<MINBAND>\t%.0f", p.MinBand)
	wl("<MAXBAND>\t%.0f", p.MaxBand)
	wl("<MEASUREMENT-CONTACT>\t%s", or(p.MeasurementContact, "#"))
	wl("<MEASUREMENT-EMAIL>\t%s", or(p.MeasurementEmail, "#"))
	wl("<MEASUREMENT-DATE>\t%s", dateStr)
	wl("<MEASUREMENT-NOTE>\t%s", or(p.MeasurementNote, "Exported from GLL by gll-tools"))
	wl("<MEASUREMENT-ENVIRONMENT>\t%s", or(p.MeasurementEnvironment, "#"))
	wl("<MEASUREMENT-DISTANCE>\t%.1f", p.MeasurementDistance)
	wl("<MEASUREMENT-INPUTVOLTAGE>\t%s", or(p.MeasurementVoltage, "0"))
	wl("<TYPE>\t<%s>", or(p.SpeakerType, "passive"))

	// Sensitivity
	if len(p.Sensitivity) > 0 {
		vals := make([]string, len(p.Sensitivity))
		for i, v := range p.Sensitivity {
			vals[i] = fmt.Sprintf("%.2f", v)
		}
		wl("<SENSITIVITY>\t%s", strings.Join(vals, "\t"))
	} else {
		wl("<SENSITIVITY>\t")
	}
	wl("<SENSITIVITY-INFO>\t#")

	wl("<IMPEDANCE>\t%.0f", p.Impedance)
	wl("<IMPEDANCE-INFO>\t#")
	wl("<TOTMAXINPUT>\t0\t0")
	wl("<RADIATION>\t<%s>", or(p.Radiation, defaultRadiation))

	// Axial spectrum
	if len(p.AxialSpectrum) > 0 {
		vals := make([]string, len(p.AxialSpectrum))
		for i, v := range p.AxialSpectrum {
			vals[i] = fmt.Sprintf("%.2f", v)
		}
		wl("<AXIAL-SPECTRUM>\t%s", strings.Join(vals, "\t"))
	} else {
		wl("<AXIAL-SPECTRUM>\t")
	}
	wl("<AXIAL-SPECTRUM-INFO>\tat 1m")

	wl("<BALLOON-SYMMETRY>\t%s", or(p.Symmetry, clfTagNone))
	wl("<BALLOON-ARC-ORDER>\t<default>")
	wl("<BALLOON-REF>\t<%s>\t<SIGN>\t<actual>", or(p.Reference, defaultReference))

	// Band data
	for f, freq := range p.BandFrequencies {
		wl("<BAND>\t%.0f", freq)

		for a := 0; a < p.AzimuthCount && a < len(p.PolarLevels); a++ {
			vals := make([]string, 0, p.PolarCount)
			for pol := 0; pol < p.PolarCount && pol < len(p.PolarLevels[a]); pol++ {
				level := 0.0
				if f < len(p.PolarLevels[a][pol]) {
					level = p.PolarLevels[a][pol][f]
				}
				vals = append(vals, fmt.Sprintf("%.2f", level))
			}
			fmt.Fprintf(w, "%s\n", strings.Join(vals, "\t"))
		}
	}

	// Footer
	if p.CabinetDXF != "" {
		wl("<CABINET>\t<dxf>%s", p.CabinetDXF)
	}
	if p.CabinetSystem != "" {
		wl("<CABINET-SYSTEM>\t%s", p.CabinetSystem)
	} else {
		wl("<CABINET-SYSTEM>")
	}
	wl("%s", endTag)

	return nil
}
