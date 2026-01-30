package cmd

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cwbudde/gll-tools/internal/clf"
	"github.com/cwbudde/gll-tools/internal/frd"
	"github.com/cwbudde/gll-tools/internal/mesh"
	"github.com/cwbudde/gll-tools/internal/viz"
	"github.com/cwbudde/gll-tools/pkg/gll"
	"github.com/spf13/cobra"
)

var (
	sourceIndex   int
	loadResponses bool
	maxResponses  int
	exportCSV     string
	exportCLF     string
	exportFRD     string
	cabinetDXF    string
)

var acousticCmd = &cobra.Command{
	Use:   "acoustic [file.gll]",
	Short: "Display acoustic data from a GLL file",
	Long: `Display detailed acoustic data including source definitions,
balloon directivity data, and frequency response information.

Examples:
  gllinfo acoustic speaker.gll                    # Show all sources
  gllinfo acoustic speaker.gll --source 0         # Show first source in detail
  gllinfo acoustic speaker.gll -s 0 --responses   # Include response data
  gllinfo acoustic speaker.gll -s 0 --export-csv output.csv  # Export to CSV
  gllinfo acoustic speaker.gll -s 0 --export-frd output/     # Export to FRD files`,
	Args: cobra.ExactArgs(1),
	RunE: runAcoustic,
}

func init() {
	rootCmd.AddCommand(acousticCmd)

	acousticCmd.Flags().IntVarP(&sourceIndex, "source", "s", -1, "source index to display (default: all)")
	acousticCmd.Flags().BoolVarP(&loadResponses, "responses", "r", false, "load and display response data")
	acousticCmd.Flags().IntVar(&maxResponses, "max-responses", 10, "maximum responses to display (default: 10)")
	acousticCmd.Flags().StringVar(&exportCSV, "export-csv", "", "export response data to CSV file")
	acousticCmd.Flags().StringVar(&exportCLF, "export-clf", "", "export directivity to CLF text format file")
	acousticCmd.Flags().StringVar(&exportFRD, "export-frd", "", "export responses to FRD files (directory path)")
	acousticCmd.Flags().StringVar(&cabinetDXF, "cabinet-dxf", "", "DXF file path to reference in CLF <CABINET> tag")
}

func runAcoustic(cmd *cobra.Command, args []string) error {
	filename := args[0]

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Parse the GLL file
	file, err := gll.Parse(f)
	if err != nil {
		return fmt.Errorf("failed to parse GLL file: %w", err)
	}

	if file.Database == nil || len(file.Database.SourceDefinitions) == 0 {
		return fmt.Errorf("no source definitions found in file")
	}

	sources := file.Database.SourceDefinitions
	placementsByDef := collectSourcePlacements(file.Database)

	// If specific source requested
	if sourceIndex >= 0 {
		if sourceIndex >= len(sources) {
			return fmt.Errorf("source index %d out of range (0-%d)", sourceIndex, len(sources)-1)
		}

		// Handle CSV export
		if exportCSV != "" {
			return exportResponsesCSV(f, sources[sourceIndex], exportCSV)
		}

		// Handle CLF export
		if exportCLF != "" {
			return exportSourceCLF(f, sources[sourceIndex], &file.GenSystem, exportCLF, file.Database.BoxTypes)
		}

		// Handle FRD export
		if exportFRD != "" {
			return exportResponsesFRD(f, sources[sourceIndex], filename, exportFRD)
		}

		return displaySource(f, sources[sourceIndex], placementsByDef, loadResponses, maxResponses)
	}

	// CLF/CSV/FRD export requires a specific source
	if exportCLF != "" {
		return fmt.Errorf("--export-clf requires --source to specify which source to export")
	}

	if exportCSV != "" {
		return fmt.Errorf("--export-csv requires --source to specify which source to export")
	}

	if exportFRD != "" {
		return fmt.Errorf("--export-frd requires --source to specify which source to export")
	}

	// Display all sources summary
	fmt.Printf("Acoustic Sources in %s:\n\n", filename)

	for i, src := range sources {
		if src.Definition == nil {
			fmt.Printf("[%d] %s (no definition)\n", i, src.Key)
			continue
		}

		def := src.Definition
		fmt.Printf("[%d] %s\n", i, def.Label)
		fmt.Printf("    Key: %s\n", src.Key)
		fmt.Printf("    Company: %s\n", def.CompanyLabel)
		fmt.Printf("    Bandwidth: %.0f - %.0f Hz\n", def.NominalBandwidthFrom, def.NominalBandwidthTo)
		fmt.Printf("    Data Type: %s\n", def.DataType)
		if placements := placementsByDef[src.Key]; len(placements) > 0 {
			fmt.Printf("    Placements: %d\n", len(placements))
		}

		if def.BalloonData != nil {
			balloon := def.BalloonData

			fmt.Printf("    Balloon:\n")
			fmt.Printf("      Symmetry: %s\n", symmetryName(balloon.AngularResolution.Symmetry))
			fmt.Printf("      Resolution: %.1f° x %.1f°\n",
				balloon.AngularResolution.MeridianStep,
				balloon.AngularResolution.ParallelStep)
			fmt.Printf("      Responses: %d\n", balloon.ResponseCount)
		}

		fmt.Printf("    Measurement: %.3fV @ %.1fm\n", def.MeasuredVoltage, def.MeasuredDistance)
		fmt.Printf("    Environment: %.1f°C, %.0f%% RH, %.2f kPa\n",
			def.Temperature, def.Humidity, def.AtmosphericPressure)
		fmt.Println()
	}

	return nil
}

func displaySource(f *os.File, src gll.SourceDefinitionItem, placementsByDef map[string][]sourcePlacement, loadResp bool, maxResp int) error {
	if src.Definition == nil {
		return fmt.Errorf("source has no definition")
	}

	def := src.Definition

	fmt.Printf("Source: %s\n", def.Label)
	fmt.Printf("Key: %s\n", src.Key)
	fmt.Printf("Company: %s\n", def.CompanyLabel)
	fmt.Printf("Bandwidth: %.0f - %.0f Hz\n", def.NominalBandwidthFrom, def.NominalBandwidthTo)
	fmt.Printf("Data Type: %s\n", def.DataType)
	fmt.Println()

	if placements := placementsByDef[src.Key]; len(placements) > 0 {
		fmt.Println("Placements:")
		for _, placement := range placements {
			fmt.Printf("  Box: %s (%s)\n", placement.BoxLabel, placement.BoxKey)
			if placement.Source.Label != "" || placement.Source.Key != "" {
				fmt.Printf("  Source: %s (%s)\n", placement.Source.Label, placement.Source.Key)
			}
			fmt.Printf("  Position: %.1f, %.1f, %.1f mm\n",
				placement.Source.Position.X,
				placement.Source.Position.Y,
				placement.Source.Position.Z)
			fmt.Printf("  Angles: H=%.2f°, V=%.2f°, R=%.2f°\n",
				radToDeg(placement.Source.Angles.X),
				radToDeg(placement.Source.Angles.Y),
				radToDeg(placement.Source.Angles.Z))
			fmt.Println()
		}
	}

	if def.BalloonData != nil {
		balloon := def.BalloonData

		fmt.Println("Balloon Data:")
		fmt.Printf("  Interpolation: %v\n", balloon.Flags&1 != 0)
		fmt.Printf("  Symmetry: %s\n", symmetryName(balloon.AngularResolution.Symmetry))
		fmt.Printf("  Front Half Only: %v\n", balloon.AngularResolution.FrontHalfOnly)
		fmt.Printf("  Meridian Step: %.1f°\n", balloon.AngularResolution.MeridianStep)
		fmt.Printf("  Parallel Step: %.1f°\n", balloon.AngularResolution.ParallelStep)
		fmt.Printf("  Grid Size: %d x %d = %d points\n",
			balloon.AngularResolution.MeridianCount(),
			balloon.AngularResolution.ParallelCount(),
			balloon.AngularResolution.TotalPoints())
		fmt.Printf("  Response Count: %d\n", balloon.ResponseCount)
		fmt.Printf("  Response Format: v%d\n", balloon.ResponseVersion)
		fmt.Println()

		// Load responses if requested
		if loadResp && balloon.ResponseCount > 0 {
			err := gll.LoadBalloonResponses(f, balloon)
			if err != nil {
				fmt.Printf("  Error loading responses: %v\n", err)
			} else {
				displayResponses(balloon, maxResp)
			}
		}
	}

	fmt.Println("\nMeasurement Conditions:")
	fmt.Printf("  Voltage: %.3f V\n", def.MeasuredVoltage)
	fmt.Printf("  Distance: %.1f m\n", def.MeasuredDistance)
	fmt.Printf("  Temperature: %.1f °C\n", def.Temperature)
	fmt.Printf("  Humidity: %.0f %%\n", def.Humidity)
	fmt.Printf("  Pressure: %.2f kPa\n", def.AtmosphericPressure)

	if def.RatedImpedance > 0 {
		fmt.Printf("  Rated Impedance: %.0f Ω\n", def.RatedImpedance)
	}

	return nil
}

func displayResponses(balloon *gll.BalloonData, maxResp int) {
	if len(balloon.Responses) == 0 {
		fmt.Println("  No responses loaded")
		return
	}

	fmt.Printf("\n  Loaded %d responses:\n", len(balloon.Responses))

	displayCount := len(balloon.Responses)
	if maxResp > 0 && displayCount > maxResp {
		displayCount = maxResp
	}

	for i := 0; i < displayCount; i++ {
		resp := balloon.Responses[i]
		fmt.Printf("\n  Response %d:\n", i)
		fmt.Printf("    Definition: %s (%d bands, %d bands/octave)\n",
			resp.Definition.GetResolutionType(),
			resp.Definition.PointCount,
			resp.Definition.BandsPerOctave)
		fmt.Printf("    Start Freq: %.1f Hz, End Freq: %.1f Hz\n",
			resp.Definition.StartFreq,
			resp.Definition.GetEndFreq())
		fmt.Printf("    Delay: %.6f s\n", resp.Delay)

		if len(resp.Level) > 0 {
			// Find min/max levels
			minLvl, maxLvl := resp.Level[0], resp.Level[0]
			for _, l := range resp.Level {
				if l < minLvl {
					minLvl = l
				}

				if l > maxLvl {
					maxLvl = l
				}
			}

			fmt.Printf("    Level Range: %.2f to %.2f dB\n", minLvl, maxLvl)
		}

		if len(resp.Phase) > 0 {
			// Find min/max phase
			minPhs, maxPhs := resp.Phase[0], resp.Phase[0]
			for _, p := range resp.Phase {
				if p < minPhs {
					minPhs = p
				}

				if p > maxPhs {
					maxPhs = p
				}
			}

			fmt.Printf("    Phase Range: %.3f to %.3f rad\n", minPhs, maxPhs)
		}
	}

	if len(balloon.Responses) > displayCount {
		fmt.Printf("\n  ... and %d more responses (use --max-responses to see more)\n",
			len(balloon.Responses)-displayCount)
	}
}

type sourcePlacement struct {
	BoxLabel string
	BoxKey   string
	Source   gll.BoxSource
}

func collectSourcePlacements(db *gll.Database) map[string][]sourcePlacement {
	placements := make(map[string][]sourcePlacement)
	if db == nil {
		return placements
	}

	for _, box := range db.BoxTypes {
		for _, src := range box.SourcePlacements {
			if src.SourceDefKey == "" {
				continue
			}
			placements[src.SourceDefKey] = append(placements[src.SourceDefKey], sourcePlacement{
				BoxLabel: box.Label,
				BoxKey:   box.Key,
				Source:   src,
			})
		}
	}

	return placements
}

func radToDeg(rad float64) float64 {
	return rad * 180 / math.Pi
}

func symmetryName(sym int32) string {
	names := []string{"None", "Vertical", "Horizontal", "Quarter", "Axial"}
	if int(sym) < len(names) {
		return names[sym]
	}

	return fmt.Sprintf("Unknown(%d)", sym)
}

// exportResponsesCSV exports all response data for a source to a CSV file
func exportResponsesCSV(f *os.File, src gll.SourceDefinitionItem, filename string) error {
	if src.Definition == nil {
		return fmt.Errorf("source has no definition")
	}

	def := src.Definition
	if def.BalloonData == nil {
		return fmt.Errorf("source has no balloon data")
	}

	balloon := def.BalloonData
	if balloon.ResponseCount == 0 {
		return fmt.Errorf("source has no responses")
	}

	// Load all responses
	err := gll.LoadBalloonResponses(f, balloon)
	if err != nil {
		return fmt.Errorf("failed to load responses: %w", err)
	}

	// Create output file
	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	// Write header
	header := []string{
		"response_index",
		"meridian_deg",
		"parallel_deg",
		"frequency_hz",
		"level_db",
		"phase_rad",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Calculate angular positions based on balloon resolution
	angRes := balloon.AngularResolution
	meridianCount := angRes.MeridianCount()
	parallelCount := angRes.ParallelCount()

	// Write data for each response
	for i, resp := range balloon.Responses {
		// Calculate meridian and parallel angles for this response
		meridianIdx := i % meridianCount
		parallelIdx := i / meridianCount

		meridianAngle := float64(meridianIdx) * angRes.MeridianStep
		parallelAngle := float64(parallelIdx) * angRes.ParallelStep

		// Sanity check for grid bounds
		if parallelIdx >= parallelCount {
			continue
		}

		// Write each frequency point
		for j := 0; j < int(resp.Definition.PointCount); j++ {
			freq := resp.Definition.GetFrequency(j)

			var level, phase float64
			if j < len(resp.Level) {
				level = resp.Level[j]
			}
			if j < len(resp.Phase) {
				phase = resp.Phase[j]
			}

			row := []string{
				strconv.Itoa(i),
				strconv.FormatFloat(meridianAngle, 'f', 1, 64),
				strconv.FormatFloat(parallelAngle, 'f', 1, 64),
				strconv.FormatFloat(freq, 'f', 2, 64),
				strconv.FormatFloat(level, 'f', 4, 64),
				strconv.FormatFloat(phase, 'f', 6, 64),
			}
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("failed to write CSV row: %w", err)
			}
		}
	}

	fmt.Printf("Exported %d responses to %s\n", len(balloon.Responses), filename)
	return nil
}

// exportResponsesFRD exports each balloon response as a separate FRD file into the given directory.
func exportResponsesFRD(f *os.File, src gll.SourceDefinitionItem, gllFilename string, outDir string) error {
	if src.Definition == nil {
		return fmt.Errorf("source has no definition")
	}

	def := src.Definition
	if def.BalloonData == nil {
		return fmt.Errorf("source has no balloon data")
	}

	balloon := def.BalloonData
	if balloon.ResponseCount == 0 {
		return fmt.Errorf("source has no responses")
	}

	if err := gll.LoadBalloonResponses(f, balloon); err != nil {
		return fmt.Errorf("failed to load responses: %w", err)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Extract basename from GLL filename (without extension)
	basename := strings.TrimSuffix(filepath.Base(gllFilename), filepath.Ext(gllFilename))

	angRes := balloon.AngularResolution
	meridianCount := angRes.MeridianCount()
	parallelCount := angRes.ParallelCount()

	for i, resp := range balloon.Responses {
		meridianIdx := i % meridianCount
		parallelIdx := i / meridianCount
		if parallelIdx >= parallelCount {
			continue
		}

		meridianAngle := float64(meridianIdx) * angRes.MeridianStep
		parallelAngle := float64(parallelIdx) * angRes.ParallelStep

		filename := filepath.Join(outDir, fmt.Sprintf("%s_m%03.0f_p%03.0f.frd", basename, meridianAngle, parallelAngle))

		frequencies := make([]float64, int(resp.Definition.PointCount))
		levels := make([]float64, int(resp.Definition.PointCount))
		phases := make([]float64, int(resp.Definition.PointCount))

		for j := range frequencies {
			frequencies[j] = resp.Definition.GetFrequency(j)
			if j < len(resp.Level) {
				levels[j] = resp.Level[j]
			}
			if j < len(resp.Phase) {
				phases[j] = resp.Phase[j]
			}
		}

		outFile, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create FRD file: %w", err)
		}

		err = frd.WriteResponse(outFile, frequencies, levels, phases)
		outFile.Close()
		if err != nil {
			return fmt.Errorf("failed to write FRD data: %w", err)
		}
	}

	fmt.Printf("Exported %d responses as FRD files to %s\n", len(balloon.Responses), outDir)
	return nil
}

// exportSourceCLF exports directivity data for a source to a CLF text format file.
func exportSourceCLF(f *os.File, src gll.SourceDefinitionItem, gen *gll.GenSystem, filename string, boxes []gll.BoxType) error {
	if src.Definition == nil {
		return fmt.Errorf("source has no definition")
	}

	def := src.Definition
	if def.BalloonData == nil {
		return fmt.Errorf("source has no balloon data")
	}

	balloon := def.BalloonData
	if balloon.ResponseCount == 0 {
		return fmt.Errorf("source has no responses")
	}

	err := gll.LoadBalloonResponses(f, balloon)
	if err != nil {
		return fmt.Errorf("failed to load responses: %w", err)
	}

	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	opts := []clf.ExportOption{
		clf.WithBoxTypes(boxes),
		clf.WithSourceKey(src.Key),
	}

	// Determine DXF path: explicit flag, or auto-generate alongside CLF.
	dxfPath := cabinetDXF
	if dxfPath == "" {
		dxfPath = autoExportCabinetDXF(filename, src.Key, boxes)
	}
	if dxfPath != "" {
		opts = append(opts, clf.WithCabinetDXF(filepath.Base(dxfPath)))
	}

	err = clf.ExportSource(outFile, def, gen, opts...)
	if err != nil {
		return fmt.Errorf("CLF export failed: %w", err)
	}

	fmt.Printf("Exported CLF text file to %s\n", filename)
	return nil
}

// autoExportCabinetDXF finds the first box containing sourceKey that has geometry,
// exports it as a DXF file alongside the CLF file, and returns the DXF path.
// Returns "" if no suitable geometry is found.
func autoExportCabinetDXF(clfPath string, sourceKey string, boxes []gll.BoxType) string {
	if sourceKey == "" {
		return ""
	}

	var geom *gll.CaseGeometry
	for _, box := range boxes {
		for _, sp := range box.SourcePlacements {
			if sp.SourceDefKey == sourceKey {
				if box.CaseGeometry != nil && len(box.CaseGeometry.Vertices) > 0 {
					geom = box.CaseGeometry
				}
				break
			}
		}
		if geom != nil {
			break
		}
	}

	if geom == nil {
		return ""
	}

	m, err := viz.BuildCaseGeometryMesh(geom)
	if err != nil {
		return ""
	}

	dxfPath := strings.TrimSuffix(clfPath, filepath.Ext(clfPath)) + ".dxf"
	dxfFile, err := os.Create(dxfPath)
	if err != nil {
		return ""
	}
	defer dxfFile.Close()

	if err := mesh.WriteDXF(dxfFile, m, "cabinet"); err != nil {
		return ""
	}

	fmt.Printf("Exported cabinet DXF to %s\n", dxfPath)
	return dxfPath
}
