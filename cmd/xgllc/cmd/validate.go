package cmd

import (
	"fmt"

	"github.com/MeKo-Christian/gll-tools/pkg/xgll"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <file.xgll>",
	Short: "Validate system-specific block constraints",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		if err := ensureFileExists(path); err != nil {
			return err
		}

		doc, err := xgll.ParseFile(path)
		systemDiags := xgll.ValidateSystemConstraints(doc)
		dataDiags := xgll.ValidateDataConstraints(doc)
		if jsonOut {
			payload := struct {
				Document          *xgll.Document    `json:"document"`
				SystemDiagnostics []xgll.Diagnostic `json:"system_diagnostics"`
				DataDiagnostics   []xgll.Diagnostic `json:"data_diagnostics"`
			}{
				Document:          doc,
				SystemDiagnostics: systemDiags,
				DataDiagnostics:   dataDiags,
			}
			if err := writeJSON(cmd.OutOrStdout(), payload); err != nil {
				return err
			}
		} else {
			writeSummary(cmd.OutOrStdout(), path, doc)
			writeDiagnostics(cmd.OutOrStdout(), "Diagnostics", doc.Diagnostics)
		}

		if jsonOut {
			if len(systemDiags)+len(dataDiags) > 0 {
				return fmt.Errorf("validation failed with %d issues", len(systemDiags)+len(dataDiags))
			}
			return err
		}

		writeDiagnostics(cmd.OutOrStdout(), "System Constraints", systemDiags)
		writeDiagnostics(cmd.OutOrStdout(), "Data Constraints", dataDiags)

		if err != nil {
			return err
		}

		if hasErrors(systemDiags) || hasErrors(dataDiags) {
			return fmt.Errorf("validation failed with %d errors", countErrors(systemDiags)+countErrors(dataDiags))
		}
		return nil
	},
}

func hasErrors(diags []xgll.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == xgll.SeverityError {
			return true
		}
	}

	return false
}

func countErrors(diags []xgll.Diagnostic) int {
	count := 0

	for _, d := range diags {
		if d.Severity == xgll.SeverityError {
			count++
		}
	}

	return count
}
