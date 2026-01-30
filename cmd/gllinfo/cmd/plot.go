package cmd

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/cwbudde/gll-tools/internal/mesh"
	"github.com/cwbudde/gll-tools/internal/viz"
	"github.com/cwbudde/gll-tools/pkg/gll"
	"github.com/spf13/cobra"
)

var (
	polarSource      int
	polarFrequency   float64
	polarBand        int
	polarNormalize   bool
	polarStep        float64
	polarOut         string
	polarWidth       int
	polarHeight      int
	polarNoOnAxis    bool
	responseSource   int
	responseOut      string
	responseWidth    int
	responseHeight   int
	responseMeridian float64
	responseParallel float64
	responseOnAxis   bool
	responseNoOnAxis bool
	responseMode     string
	balloonSource    int
	balloonFrequency float64
	balloonBand      int
	balloonOut       string
	balloonRange     float64
	balloonScale     float64
	balloonNormalize bool
	balloonCenter    string
	geomBox          int
	geomFrame        int
	geomOut          string
	geomCenter       string
	geomPreferFrame  bool
)

var plotCmd = &cobra.Command{
	Use:   "plot",
	Short: "Generate visualizations from GLL data",
	Long: `Generate plots from acoustic data:
  gllinfo plot polar speaker.gll --source 0 --output polar.svg
  gllinfo plot response speaker.gll --source 0 --output response.svg`,
}

var plotPolarCmd = &cobra.Command{
	Use:   "polar [file.gll]",
	Short: "Generate a polar directivity plot (SVG)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlotPolar,
}

var plotResponseCmd = &cobra.Command{
	Use:   "response [file.gll]",
	Short: "Generate a frequency response plot (SVG)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlotResponse,
}

var plotBalloonCmd = &cobra.Command{
	Use:   "balloon [file.gll]",
	Short: "Generate a 3D balloon mesh (STL/OBJ)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlotBalloon,
}

var plotGeometryCmd = &cobra.Command{
	Use:   "geometry [file.gll]",
	Short: "Export cabinet or frame geometry (STL/OBJ/DXF)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlotGeometry,
}

func init() {
	rootCmd.AddCommand(plotCmd)
	plotCmd.AddCommand(plotPolarCmd)
	plotCmd.AddCommand(plotResponseCmd)
	plotCmd.AddCommand(plotBalloonCmd)
	plotCmd.AddCommand(plotGeometryCmd)

	plotPolarCmd.Flags().IntVarP(&polarSource, "source", "s", -1, "source index to plot (required)")
	plotPolarCmd.Flags().Float64VarP(&polarFrequency, "frequency", "f", 1000, "target frequency in Hz")
	plotPolarCmd.Flags().IntVar(&polarBand, "band", -1, "frequency band index (overrides --frequency)")
	plotPolarCmd.Flags().BoolVar(&polarNormalize, "normalize", false, "normalize levels to peak (per slice)")
	plotPolarCmd.Flags().Float64Var(&polarStep, "step", 10, "angle step in degrees")
	plotPolarCmd.Flags().StringVarP(&polarOut, "output", "o", "", "output SVG file path")
	plotPolarCmd.Flags().IntVar(&polarWidth, "width", 900, "output width in pixels")
	plotPolarCmd.Flags().IntVar(&polarHeight, "height", 700, "output height in pixels")
	plotPolarCmd.Flags().BoolVar(&polarNoOnAxis, "no-on-axis", false, "do not combine on-axis spectrum")

	plotResponseCmd.Flags().IntVarP(&responseSource, "source", "s", -1, "source index to plot (required)")
	plotResponseCmd.Flags().StringVarP(&responseOut, "output", "o", "", "output SVG file path")
	plotResponseCmd.Flags().IntVar(&responseWidth, "width", 1000, "output width in pixels")
	plotResponseCmd.Flags().IntVar(&responseHeight, "height", 700, "output height in pixels")
	plotResponseCmd.Flags().Float64Var(&responseMeridian, "meridian", 0, "meridian angle in degrees (0=top, 90=right)")
	plotResponseCmd.Flags().Float64Var(&responseParallel, "parallel", 0, "parallel angle in degrees (0=front/on-axis)")
	plotResponseCmd.Flags().BoolVar(&responseOnAxis, "on-axis", false, "use on-axis spectrum instead of balloon response")
	plotResponseCmd.Flags().BoolVar(&responseNoOnAxis, "no-on-axis", false, "do not combine on-axis spectrum")
	plotResponseCmd.Flags().StringVar(&responseMode, "mode", "magnitude", "response plot mode: magnitude, phase-wrapped, phase-unwrapped, group-delay")

	plotBalloonCmd.Flags().IntVarP(&balloonSource, "source", "s", -1, "source index to plot (required)")
	plotBalloonCmd.Flags().Float64VarP(&balloonFrequency, "frequency", "f", 1000, "target frequency in Hz")
	plotBalloonCmd.Flags().IntVar(&balloonBand, "band", -1, "frequency band index (overrides --frequency)")
	plotBalloonCmd.Flags().StringVarP(&balloonOut, "output", "o", "", "output STL file path")
	plotBalloonCmd.Flags().Float64Var(&balloonRange, "range", 40, "dB range for radial scaling")
	plotBalloonCmd.Flags().Float64Var(&balloonScale, "scale", 1, "overall size scaling factor")
	plotBalloonCmd.Flags().BoolVar(&balloonNormalize, "normalize", false, "normalize against local max at frequency")
	plotBalloonCmd.Flags().StringVar(&balloonCenter, "center", "origin", "center mesh at: origin, bbox, centroid")

	plotGeometryCmd.Flags().IntVar(&geomBox, "box", -1, "box index to export (default: 0)")
	plotGeometryCmd.Flags().IntVar(&geomFrame, "frame", -1, "frame index to export")
	plotGeometryCmd.Flags().BoolVar(&geomPreferFrame, "prefer-frame", false, "use frame geometry when both are present")
	plotGeometryCmd.Flags().StringVarP(&geomOut, "output", "o", "", "output STL/OBJ/DXF file path")
	plotGeometryCmd.Flags().StringVar(&geomCenter, "center", "origin", "center mesh at: origin, bbox, centroid")
}

func runPlotPolar(cmd *cobra.Command, args []string) error {
	if polarSource < 0 {
		return fmt.Errorf("--source is required")
	}
	if polarOut == "" {
		return fmt.Errorf("--output is required")
	}
	if err := ensureSVGOutput(polarOut); err != nil {
		return err
	}

	file, f, err := loadGLL(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	if file.Database == nil || polarSource >= len(file.Database.SourceDefinitions) {
		return fmt.Errorf("source index %d out of range", polarSource)
	}

	src := file.Database.SourceDefinitions[polarSource]
	if src.Definition == nil || src.Definition.BalloonData == nil {
		return fmt.Errorf("source %d has no balloon data", polarSource)
	}

	if err := gll.LoadBalloonResponses(f, src.Definition.BalloonData); err != nil {
		return fmt.Errorf("loading balloon responses: %w", err)
	}

	if len(src.Definition.BalloonData.Responses) == 0 {
		return fmt.Errorf("no balloon responses available")
	}

	respDef := src.Definition.BalloonData.Responses[0].Definition
	freqs := viz.BuildFrequencyList(respDef)
	if len(freqs) == 0 {
		return fmt.Errorf("unable to build frequency grid")
	}

	freqIndex := polarBand
	if freqIndex < 0 {
		freqIndex = viz.FindNearestFrequencyIndex(freqs, polarFrequency)
	}
	if freqIndex < 0 || freqIndex >= len(freqs) {
		return fmt.Errorf("frequency index %d out of range", freqIndex)
	}

	slices, err := viz.ComputePolarSlices(src.Definition, freqIndex, polarStep, !polarNoOnAxis)
	if err != nil {
		return err
	}

	if polarNormalize {
		normalizeLevels(slices.HorizontalLevel)
		normalizeLevels(slices.VerticalLevel)
	}

	out, err := os.Create(polarOut)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	title := src.Definition.Label
	if title == "" {
		title = src.Key
	}
	plot := viz.PolarPlot{
		Width:       polarWidth,
		Height:      polarHeight,
		Title:       title,
		FrequencyHz: slices.FrequencyHz,
		AnglesDeg:   slices.AnglesDeg,
		Horizontal:  slices.HorizontalLevel,
		Vertical:    slices.VerticalLevel,
		Normalize:   polarNormalize,
		UsesOnAxis:  slices.UsesOnAxis,
	}

	if err := viz.RenderPolarSVG(out, plot); err != nil {
		return err
	}

	return nil
}

func runPlotResponse(cmd *cobra.Command, args []string) error {
	if responseSource < 0 {
		return fmt.Errorf("--source is required")
	}
	if responseOut == "" {
		return fmt.Errorf("--output is required")
	}
	if err := ensureSVGOutput(responseOut); err != nil {
		return err
	}

	file, f, err := loadGLL(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	if file.Database == nil || responseSource >= len(file.Database.SourceDefinitions) {
		return fmt.Errorf("source index %d out of range", responseSource)
	}

	src := file.Database.SourceDefinitions[responseSource]
	if src.Definition == nil {
		return fmt.Errorf("source %d has no definition", responseSource)
	}

	var response *gll.TransferFunction
	combineOnAxis := !responseNoOnAxis
	if responseOnAxis || src.Definition.BalloonData == nil {
		response = src.Definition.OnAxisSpectrum
		combineOnAxis = false
		if responseOnAxis && response == nil {
			return fmt.Errorf("on-axis spectrum not available")
		}
	} else {
		if err := gll.LoadBalloonResponses(f, src.Definition.BalloonData); err != nil {
			return fmt.Errorf("loading balloon responses: %w", err)
		}
		response = viz.ResponseAtAngles(src.Definition.BalloonData, responseMeridian, responseParallel)
		if response == nil {
			return fmt.Errorf("no response at meridian %.1f, parallel %.1f", responseMeridian, responseParallel)
		}
	}

	if response == nil {
		return fmt.Errorf("no response data available")
	}

	series, err := viz.BuildResponseSeries(src.Definition, response, combineOnAxis)
	if err != nil {
		return err
	}

	plotKind, plotSeries, err := selectResponseSeries(series, responseMode)
	if err != nil {
		return err
	}

	out, err := os.Create(responseOut)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	title := src.Definition.Label
	if title == "" {
		title = src.Key
	}
	plot := viz.ResponsePlot{
		Width:       responseWidth,
		Height:      responseHeight,
		Title:       title,
		Frequencies: series.Frequencies,
		Series:      plotSeries,
		Kind:        plotKind,
		UsesOnAxis:  series.UsesOnAxis,
	}

	if err := viz.RenderResponseSVG(out, plot); err != nil {
		return err
	}

	return nil
}

func runPlotBalloon(cmd *cobra.Command, args []string) error {
	if balloonSource < 0 {
		return fmt.Errorf("--source is required")
	}
	if balloonOut == "" {
		return fmt.Errorf("--output is required")
	}
	if err := ensureMeshOutput(balloonOut); err != nil {
		return err
	}

	file, f, err := loadGLL(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	if file.Database == nil || balloonSource >= len(file.Database.SourceDefinitions) {
		return fmt.Errorf("source index %d out of range", balloonSource)
	}

	src := file.Database.SourceDefinitions[balloonSource]
	if src.Definition == nil || src.Definition.BalloonData == nil {
		return fmt.Errorf("source %d has no balloon data", balloonSource)
	}

	if err := gll.LoadBalloonResponses(f, src.Definition.BalloonData); err != nil {
		return fmt.Errorf("loading balloon responses: %w", err)
	}
	if len(src.Definition.BalloonData.Responses) == 0 {
		return fmt.Errorf("no balloon responses available")
	}

	respDef := src.Definition.BalloonData.Responses[0].Definition
	freqs := viz.BuildFrequencyList(respDef)
	if len(freqs) == 0 {
		return fmt.Errorf("unable to build frequency grid")
	}

	freqIndex := balloonBand
	if freqIndex < 0 {
		freqIndex = viz.FindNearestFrequencyIndex(freqs, balloonFrequency)
	}
	if freqIndex < 0 || freqIndex >= len(freqs) {
		return fmt.Errorf("frequency index %d out of range", freqIndex)
	}

	mesh, err := viz.BuildBalloonMesh(src.Definition, freqIndex, balloonRange, balloonScale, balloonNormalize)
	if err != nil {
		return err
	}
	if err := applyMeshCenter(mesh, balloonCenter); err != nil {
		return err
	}

	out, err := os.Create(balloonOut)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	name := src.Definition.Label
	if name == "" {
		name = src.Key
	}
	return writeMeshFile(balloonOut, mesh, name)
}

func loadGLL(path string) (*gll.File, *os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	file, err := gll.Parse(f)
	if err != nil {
		f.Close()
		return nil, nil, fmt.Errorf("failed to parse GLL file: %w", err)
	}
	return file, f, nil
}

func ensureSVGOutput(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return fmt.Errorf("output path must have .svg extension")
	}
	if ext != ".svg" {
		return fmt.Errorf("only .svg output is supported for now")
	}
	return nil
}

func runPlotGeometry(cmd *cobra.Command, args []string) error {
	if geomOut == "" {
		return fmt.Errorf("--output is required")
	}
	if err := ensureMeshOutput(geomOut); err != nil {
		return err
	}

	file, f, err := loadGLL(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	if file.Database == nil {
		return fmt.Errorf("no database found")
	}

	geom, label, err := selectCaseGeometry(file.Database, geomBox, geomFrame, geomPreferFrame)
	if err != nil {
		return err
	}

	mesh, err := viz.BuildCaseGeometryMesh(geom)
	if err != nil {
		return err
	}
	if err := applyMeshCenter(mesh, geomCenter); err != nil {
		return err
	}

	return writeMeshFile(geomOut, mesh, label)
}

func ensureMeshOutput(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return fmt.Errorf("output path must have .stl, .obj, or .dxf extension")
	}
	if ext != ".stl" && ext != ".obj" && ext != ".dxf" {
		return fmt.Errorf("supported output formats: .stl, .obj, .dxf")
	}
	return nil
}

func applyMeshCenter(meshData *mesh.Mesh, center string) error {
	mode := strings.ToLower(strings.TrimSpace(center))
	if mode == "" {
		return nil
	}
	switch mode {
	case string(mesh.CenterOrigin), string(mesh.CenterBBox), string(mesh.CenterCentroid):
		mesh.CenterMesh(meshData, mesh.CenterMode(mode))
		return nil
	default:
		return fmt.Errorf("unknown center mode %q (use origin, bbox, centroid)", center)
	}
}

func writeMeshFile(path string, meshData *mesh.Mesh, name string) error {
	ext := strings.ToLower(filepath.Ext(path))
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	switch ext {
	case ".stl":
		if len(meshData.Indices) == 0 {
			return fmt.Errorf("no face data available for STL output")
		}
		return mesh.WriteSTL(out, meshData, name)
	case ".obj":
		mtlFilename := mesh.SanitizeMTLFilename(strings.TrimSuffix(filepath.Base(path), ext))
		mtlPath := filepath.Join(filepath.Dir(path), mtlFilename)
		mtlFile, err := os.Create(mtlPath)
		if err != nil {
			return fmt.Errorf("creating mtl file: %w", err)
		}
		if err := mesh.WriteMTL(mtlFile, name); err != nil {
			_ = mtlFile.Close()
			return err
		}
		if err := mtlFile.Close(); err != nil {
			return err
		}
		return mesh.WriteOBJ(out, meshData, name, mtlFilename)
	case ".dxf":
		return mesh.WriteDXF(out, meshData, name)
	default:
		return fmt.Errorf("unsupported mesh output format")
	}
}

func selectCaseGeometry(db *gll.Database, boxIndex, frameIndex int, preferFrame bool) (*gll.CaseGeometry, string, error) {
	if db == nil {
		return nil, "", fmt.Errorf("no database")
	}

	if frameIndex >= 0 {
		if frameIndex >= len(db.Frames) {
			return nil, "", fmt.Errorf("frame index %d out of range", frameIndex)
		}
		frame := db.Frames[frameIndex]
		if frame.CaseGeometry == nil {
			return nil, "", fmt.Errorf("frame %d has no geometry", frameIndex)
		}
		label := frame.Label
		if label == "" {
			label = frame.Key
		}
		return frame.CaseGeometry, label, nil
	}

	if preferFrame && len(db.Frames) > 0 {
		for _, frame := range db.Frames {
			if frame.CaseGeometry == nil {
				continue
			}
			label := frame.Label
			if label == "" {
				label = frame.Key
			}
			return frame.CaseGeometry, label, nil
		}
	}

	if len(db.BoxTypes) == 0 {
		return nil, "", fmt.Errorf("no box types available")
	}
	if boxIndex < 0 {
		boxIndex = 0
	}
	if boxIndex >= len(db.BoxTypes) {
		return nil, "", fmt.Errorf("box index %d out of range", boxIndex)
	}
	box := db.BoxTypes[boxIndex]
	if box.CaseGeometry == nil {
		return nil, "", fmt.Errorf("box %d has no geometry", boxIndex)
	}
	label := box.Label
	if label == "" {
		label = box.Key
	}
	return box.CaseGeometry, label, nil
}

func normalizeLevels(values []float64) {
	maxVal := math.NaN()
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		if math.IsNaN(maxVal) || v > maxVal {
			maxVal = v
		}
	}
	if math.IsNaN(maxVal) {
		return
	}
	for i, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		values[i] = v - maxVal
	}
}

func selectResponseSeries(series *viz.ResponseSeries, mode string) (viz.ResponsePlotKind, []float64, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "magnitude", "level":
		return viz.ResponseMagnitude, series.Level, nil
	case "phase-wrapped", "wrapped":
		return viz.ResponsePhaseWrapped, series.PhaseWrapped, nil
	case "phase-unwrapped", "unwrapped", "phase":
		return viz.ResponsePhaseUnwrap, series.Phase, nil
	case "group-delay", "delay":
		return viz.ResponseGroupDelay, series.GroupDelayMs, nil
	default:
		return "", nil, fmt.Errorf("unknown mode %q (use magnitude, phase-wrapped, phase-unwrapped, group-delay)", mode)
	}
}
