package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cwbudde/gll-tools/pkg/xgll"
	"github.com/spf13/cobra"
)

var (
	convertOutput string
	convertFormat string
	convertPretty bool
)

var convertCmd = &cobra.Command{
	Use:   "convert <file.xgll>",
	Short: "Convert an XGLL file to binary",
	Long:  "Convert an XGLL file to a binary format. The GLL writer currently emits a minimal file (header + GenSystem + empty database).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		if err := ensureFileExists(path); err != nil {
			return err
		}
		if convertOutput == "" {
			return fmt.Errorf("missing --output")
		}

		doc, err := xgll.ParseFile(path)
		if err != nil {
			return err
		}

		format := convertFormat
		if convertPretty {
			if strings.EqualFold(format, "xgllbin") {
				format = "xgllbin-pretty"
			} else {
				return fmt.Errorf("--pretty is only supported for xgllbin output")
			}
		}

		writer, err := xgll.GetWriter(format)
		if err != nil {
			return fmt.Errorf("%w (supported: %s)", err, strings.Join(xgll.ListWriterFormats(), ", "))
		}

		if err := os.MkdirAll(filepath.Dir(convertOutput), 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}

		out, err := os.Create(convertOutput)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer out.Close()

		if err := writer.Write(doc, out); err != nil {
			_ = out.Close()
			_ = os.Remove(convertOutput)
			return err
		}

		return nil
	},
}

func init() {
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "", "output file path")
	convertCmd.Flags().StringVarP(&convertFormat, "format", "f", "xgllbin", "output format (xgllbin, gll)")
	convertCmd.Flags().BoolVar(&convertPretty, "pretty", false, "use pretty-printed JSON payload for xgllbin")
}
