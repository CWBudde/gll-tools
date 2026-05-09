package cmd

import (
	"fmt"

	"github.com/cwbudde/gll-tools/pkg/sofaexport"
	"github.com/spf13/cobra"
)

var exportSofaCmd = &cobra.Command{
	Use:   "sofa [file.gll]",
	Short: "Export directivity balloons to SOFA (FreeFieldDirectivityTF)",
	Long: `Export each SourceDefinition's BalloonData to a SOFA file using the
FreeFieldDirectivityTF convention (AES69-2015).

By default, each direction's transfer function is the source's absolute
on-axis spectrum multiplied by the relative balloon directivity. Use
--relative to emit the raw balloon TF without the on-axis combine.

The output is one .sofa file per BalloonData, named according to --pattern
(default "{gll}__{source}__{usecase}.sofa") in --output-dir.

Examples:
  gllinfo export sofa speaker.gll                          # write to ./
  gllinfo export sofa speaker.gll -o sofa_out -v
  gllinfo export sofa speaker.gll --relative
  gllinfo export sofa speaker.gll --source LF --overwrite`,
	Args: cobra.ExactArgs(1),
	RunE: runExportSofa,
}

func init() {
	exportCmd.AddCommand(exportSofaCmd)

	exportSofaCmd.Flags().StringP("output-dir", "o", ".", "directory for .sofa output files")
	exportSofaCmd.Flags().Bool("relative", false, "emit raw balloon TF (skip on-axis combine)")
	exportSofaCmd.Flags().String("pattern", "{gll}__{source}__{usecase}.sofa", "filename template ({gll}, {source}, {usecase})")
	exportSofaCmd.Flags().String("source", "", "only export the named source (key or label)")
	exportSofaCmd.Flags().String("use-case", "", "only export the named use case")
	exportSofaCmd.Flags().Bool("overwrite", false, "overwrite existing output files")
	exportSofaCmd.Flags().BoolP("verbose", "v", false, "print one line per produced file")
}

func runExportSofa(cmd *cobra.Command, args []string) error {
	outputDir, _ := cmd.Flags().GetString("output-dir")
	relative, _ := cmd.Flags().GetBool("relative")
	pattern, _ := cmd.Flags().GetString("pattern")
	sourceFilter, _ := cmd.Flags().GetString("source")
	useCaseFilter, _ := cmd.Flags().GetString("use-case")
	overwrite, _ := cmd.Flags().GetBool("overwrite")
	verbose, _ := cmd.Flags().GetBool("verbose")

	opts := sofaexport.Options{
		Relative:        relative,
		OutputDir:       outputDir,
		FilenamePattern: pattern,
		SourceFilter:    sourceFilter,
		UseCaseFilter:   useCaseFilter,
		Overwrite:       overwrite,
	}

	paths, err := sofaexport.ExportFile(args[0], opts)
	if err != nil {
		return err
	}
	if verbose {
		for _, p := range paths {
			fmt.Println(p)
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %d SOFA file(s)\n", len(paths))
	return nil
}
