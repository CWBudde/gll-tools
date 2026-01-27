package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var jsonOut bool

var rootCmd = &cobra.Command{
	Use:   "xgllc",
	Short: "Parse and convert EASE XGLL text files",
	Long: `xgllc parses EASE SpeakerLab XGLL text files and can convert them
to a binary representation. Conversion is planned but not yet implemented.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format")
	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(convertCmd)
}

func ensureFileExists(path string) error {
	if path == "" {
		return fmt.Errorf("missing file path")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}

	return nil
}
