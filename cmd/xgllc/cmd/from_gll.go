package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cwbudde/gll-tools/pkg/gll"
	"github.com/cwbudde/gll-tools/pkg/xgll"
	"github.com/spf13/cobra"
)

var fromGLLCmd = &cobra.Command{
	Use:   "from-gll <file.gll>",
	Short: "Convert a GLL file to XGLL text",
	Long: `Convert a GLL file to XGLL text. The current converter emits a minimal
System header and empty Layout/Data blocks based on the GLL GenSystem.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		if err := ensureFileExists(path); err != nil {
			return err
		}

		output, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening file: %w", err)
		}
		defer f.Close()

		file, err := gll.Parse(f)
		if err != nil {
			return fmt.Errorf("parsing GLL file: %w", err)
		}

		doc, err := xgll.BuildXGLLDocument(file)
		if err != nil {
			return fmt.Errorf("building XGLL: %w", err)
		}

		if output == "" {
			return xgll.WriteXGLL(doc, os.Stdout)
		}

		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}

		out, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer out.Close()

		return xgll.WriteXGLL(doc, out)
	},
}

func init() {
	fromGLLCmd.Flags().StringP("output", "o", "", "output .xgll file (defaults to stdout)")
}
