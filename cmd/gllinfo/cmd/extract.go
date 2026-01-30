package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/cwbudde/gll-tools/pkg/gll"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var extractCmd = &cobra.Command{
	Use:   "extract [file.gll]",
	Short: "Extract embedded resources from a GLL file",
	Long: `Extract embedded resources (images, PDFs, compressed data, etc.) from a GLL file.

Resources are extracted to the current directory or a specified output directory.

Resource types:
  - DataFiles: PNG images, 3D geometry (.xed)
  - IncludeFiles: PDF documentation, technical drawings, spec sheets
  - Scanned resources: Compressed fonts, acoustic data

Examples:
  gllinfo extract speaker.gll                    # Extract all resources
  gllinfo extract --images speaker.gll           # Extract only images
  gllinfo extract --docs speaker.gll             # Extract only PDFs/documentation
  gllinfo extract --output ./extracted speaker.gll`,
	Args: cobra.ExactArgs(1),
	RunE: runExtract,
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().String("output", ".", "output directory for extracted files")
	extractCmd.Flags().Bool("images", false, "extract only images (PNG, JPG)")
	extractCmd.Flags().Bool("data", false, "extract only DataFiles")
	extractCmd.Flags().Bool("docs", false, "extract only documentation (PDFs from IncludeFiles)")
	extractCmd.Flags().Bool("decompress", true, "decompress zlib resources")

	_ = viper.BindPFlag("extract.output", extractCmd.Flags().Lookup("output"))
	_ = viper.BindPFlag("extract.images", extractCmd.Flags().Lookup("images"))
	_ = viper.BindPFlag("extract.data", extractCmd.Flags().Lookup("data"))
	_ = viper.BindPFlag("extract.docs", extractCmd.Flags().Lookup("docs"))
	_ = viper.BindPFlag("extract.decompress", extractCmd.Flags().Lookup("decompress"))
}

func runExtract(cmd *cobra.Command, args []string) error {
	ctx := &extractCtx{
		outputDir:      viper.GetString("extract.output"),
		imagesOnly:     viper.GetBool("extract.images"),
		dataOnly:       viper.GetBool("extract.data"),
		docsOnly:       viper.GetBool("extract.docs"),
		decompress:     viper.GetBool("extract.decompress"),
		extractedNames: make(map[string]bool),
	}

	// Ensure output directory exists
	if err := os.MkdirAll(ctx.outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	f, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	// Parse the GLL file
	file, err := gll.Parse(f)
	if err != nil {
		return fmt.Errorf("parsing GLL file: %w", err)
	}

	// Extract DataFiles (these are the primary source, extracted from database)
	if file.Database != nil {
		ctx.extractDataFiles(f, file)
		ctx.extractIncludeFiles(f, file)
	}

	// Extract resources (only non-PNG or PNG not already extracted via DataFiles)
	ctx.extractResources(f, file)

	fmt.Printf("\nTotal extracted: %d files\n", ctx.extractedCount)

	return nil
}

type extractCtx struct {
	outputDir      string
	imagesOnly     bool
	dataOnly       bool
	docsOnly       bool
	decompress     bool
	extractedNames map[string]bool
	extractedCount int
}

func (ctx *extractCtx) extractDataFiles(f *os.File, file *gll.File) {
	if ctx.imagesOnly || ctx.docsOnly {
		return
	}
	for _, df := range file.Database.DataFiles {
		if ctx.dataOnly || (!ctx.imagesOnly && !ctx.dataOnly && !ctx.docsOnly) {
			data, err := gll.ExtractDataFile(f, df)
			if err != nil {
				slog.Warn("failed to extract data file", "file", df.Filename, "err", err)
				continue
			}

			// Clean filename (remove leading .\ or ./ and convert to base name)
			cleanName := cleanFilename(df.Filename)

			if ctx.extractedNames[cleanName] {
				continue // Skip duplicates
			}

			outPath := filepath.Join(ctx.outputDir, cleanName)
			//nolint:gosec // G306: Extracted resources should be world-readable
			if err := os.WriteFile(outPath, data, 0o644); err != nil {
				slog.Warn("failed to write extracted data file", "path", outPath, "err", err)
				continue
			}

			fmt.Printf("Extracted: %s (%d bytes)\n", outPath, len(data))

			ctx.extractedNames[cleanName] = true
			ctx.extractedCount++
		}
	}
}

func (ctx *extractCtx) extractIncludeFiles(f *os.File, file *gll.File) {
	if ctx.imagesOnly || ctx.dataOnly {
		return
	}
	for _, inc := range file.Database.IncludeFiles {
		if ctx.docsOnly || (!ctx.imagesOnly && !ctx.dataOnly && !ctx.docsOnly) {
			data, err := gll.ExtractIncludeFile(f, inc)
			if err != nil {
				slog.Warn("failed to extract include file", "file", inc.Filename, "err", err)
				continue
			}

			// Clean filename (remove leading .\ or ./ and convert to base name)
			cleanName := cleanFilename(inc.Filename)

			if ctx.extractedNames[cleanName] {
				continue // Skip duplicates
			}

			outPath := filepath.Join(ctx.outputDir, cleanName)
			//nolint:gosec // G306: Extracted resources should be world-readable
			if err := os.WriteFile(outPath, data, 0o644); err != nil {
				slog.Warn("failed to write include file", "path", outPath, "err", err)
				continue
			}

			fmt.Printf("Extracted: %s (%d bytes) [%s]\n", outPath, len(data), inc.Label)

			ctx.extractedNames[cleanName] = true
			ctx.extractedCount++
		}
	}
}

func (ctx *extractCtx) extractResources(f *os.File, file *gll.File) {
	if ctx.dataOnly || ctx.docsOnly {
		return
	}
	for i, res := range file.Resources {
		// Filter by type if requested
		if ctx.imagesOnly && res.Type != gll.ResourceTypePNG {
			continue
		}

		var (
			data    []byte
			outName string
			err     error
		)

		switch res.Type {
		case gll.ResourceTypePNG:
			data, outName, err = ctx.processPNGResource(f, res, i)
		case gll.ResourceTypeZlib:
			data, outName, err = ctx.processZlibResource(f, res, i)
		}

		if err != nil || len(data) == 0 || outName == "" || ctx.extractedNames[outName] {
			if err != nil {
				slog.Warn("failed to process resource", "type", res.Type, "index", i, "err", err)
			}
			continue
		}

		outPath := filepath.Join(ctx.outputDir, outName)
		//nolint:gosec // G306: Extracted resources should be world-readable
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			slog.Warn("failed to write extracted resource", "path", outPath, "err", err)
			continue
		}

		fmt.Printf("Extracted: %s (%d bytes)\n", outPath, len(data))

		ctx.extractedNames[outName] = true
		ctx.extractedCount++
	}
}

func (ctx *extractCtx) processPNGResource(f *os.File, res gll.Resource, index int) ([]byte, string, error) {
	// Skip if already extracted from DataFiles
	if res.Name != "" {
		cleanName := cleanFilename(res.Name)
		if ctx.extractedNames[cleanName] {
			return nil, "", nil
		}
	}

	data, err := gll.ExtractResource(f, res)
	if err != nil {
		return nil, "", err
	}

	outName := ""
	if res.Name != "" {
		outName = cleanFilename(res.Name)
	} else {
		outName = fmt.Sprintf("resource_%d.png", index)
	}
	return data, outName, nil
}

func (ctx *extractCtx) processZlibResource(f *os.File, res gll.Resource, index int) ([]byte, string, error) {
	if ctx.decompress {
		data, err := gll.DecompressResource(f, res)
		if err != nil {
			return nil, "", err
		}

		ext := getExtensionForContent(res.Name)
		return data, fmt.Sprintf("zlib_%d%s", index, ext), nil
	}

	data, err := gll.ExtractResource(f, res)
	if err != nil {
		return nil, "", err
	}

	return data, fmt.Sprintf("zlib_%d.zlib", index), nil
}

// cleanFilename normalizes a Windows-style path to just the base filename
func cleanFilename(path string) string {
	// Replace Windows backslashes
	path = strings.ReplaceAll(path, "\\", "/")
	// Remove leading ./ or ./
	path = strings.TrimPrefix(path, "./")
	// Get base name
	return filepath.Base(path)
}

func getExtensionForContent(contentType string) string {
	switch contentType {
	case "pdf-cmap", "pdf-graphics":
		return ".pdf.txt"
	case "font-ttf":
		return ".ttf"
	case "font-data":
		return ".otf"
	case "text":
		return ".txt"
	case "acoustic-data":
		return ".dat"
	default:
		return ".bin"
	}
}
