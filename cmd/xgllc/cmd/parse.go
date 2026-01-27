package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/pkg/xgll"
	"github.com/spf13/cobra"
)

var parseCmd = &cobra.Command{
	Use:   "parse <file.xgll>",
	Short: "Parse an XGLL file and report diagnostics",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		if err := ensureFileExists(path); err != nil {
			return err
		}

		doc, err := xgll.ParseFile(path)
		if jsonOut {
			if err := writeJSON(cmd.OutOrStdout(), doc); err != nil {
				return err
			}
		} else {
			writeSummary(cmd.OutOrStdout(), path, doc)
			writeDiagnostics(cmd.OutOrStdout(), "Diagnostics", doc.Diagnostics)
		}

		return err
	},
}

func writeJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")

	return enc.Encode(v)
}

func writeSummary(out io.Writer, path string, doc *xgll.Document) {
	fmt.Fprintf(out, "File: %s\n", path)
	fmt.Fprintf(out, "Statements: %d\n", len(doc.Statements))
	fmt.Fprintf(out, "Blocks: %d\n", len(doc.Blocks))
}

func writeDiagnostics(out io.Writer, title string, diags []xgll.Diagnostic) {
	if len(diags) == 0 {
		fmt.Fprintf(out, "%s: none\n", title)
		return
	}

	fmt.Fprintf(out, "%s:\n", title)

	for _, d := range diags {
		sev := "ERROR"
		if d.Severity == xgll.SeverityWarning {
			sev = "WARN"
		}

		fmt.Fprintf(out, "  %s: line %d, col %d: %s\n", sev, d.Line, d.Column, d.Message)
	}
}
