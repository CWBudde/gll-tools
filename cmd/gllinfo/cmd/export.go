package cmd

import (
	"github.com/spf13/cobra"
)

// exportCmd is the parent command for export-format subcommands. New formats
// (CSV, FRD, etc.) are added as additional subcommands.
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export GLL data to other formats",
	Long: `Export GLL data to interchange formats.

Currently supported subcommands:
  sofa    Export directivity balloons to SOFA (FreeFieldDirectivityTF)
`,
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
