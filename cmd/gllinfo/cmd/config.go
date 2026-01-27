package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cwbudde/gll-tools/pkg/gll"
	"github.com/spf13/cobra"
)

var (
	showLimits   bool
	showWarnings bool
	showFilters  bool
	configJSON   bool
)

var configCmd = &cobra.Command{
	Use:   "config [file.gll]",
	Short: "Display configuration data from a GLL file",
	Long: `Display configuration data including mechanical limits, warnings,
and filter group definitions from a GLL file.

Examples:
  gllinfo config speaker.gll                # Show all config
  gllinfo config speaker.gll --limits       # Show only limits
  gllinfo config speaker.gll --warnings     # Show only warnings
  gllinfo config speaker.gll --filters      # Show only filter groups
  gllinfo config speaker.gll --json         # Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.Flags().BoolVar(&showLimits, "limits", false, "show mechanical/electrical limits only")
	configCmd.Flags().BoolVar(&showWarnings, "warnings", false, "show configuration warnings only")
	configCmd.Flags().BoolVar(&showFilters, "filters", false, "show filter group definitions only")
	configCmd.Flags().BoolVar(&configJSON, "json", false, "output in JSON format")
}

type configOutput struct {
	Limits       []gll.Limit       `json:"limits,omitempty"`
	Warnings     []gll.Warning     `json:"warnings,omitempty"`
	FilterGroups []gll.FilterGroup `json:"filter_groups,omitempty"`
}

func runConfig(cmd *cobra.Command, args []string) error {
	filename := args[0]

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	file, err := gll.Parse(f)
	if err != nil {
		return fmt.Errorf("failed to parse GLL file: %w", err)
	}

	if file.Database == nil {
		return fmt.Errorf("no database found in file")
	}

	db := file.Database

	// If no specific flag set, show all
	showAll := !showLimits && !showWarnings && !showFilters

	if configJSON {
		return outputConfigJSON(db, showAll)
	}

	outputConfigText(db, showAll)
	return nil
}

func outputConfigJSON(db *gll.Database, showAll bool) error {
	out := configOutput{}

	if showAll || showLimits {
		out.Limits = db.Limits
	}
	if showAll || showWarnings {
		out.Warnings = db.Warnings
	}
	if showAll || showFilters {
		out.FilterGroups = db.FilterGroups
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputConfigText(db *gll.Database, showAll bool) {
	if showAll || showLimits {
		displayLimits(db.Limits)
	}

	if showAll || showWarnings {
		displayWarnings(db.Warnings)
	}

	if showAll || showFilters {
		displayFilterGroups(db.FilterGroups)
	}
}

func displayLimits(limits []gll.Limit) {
	fmt.Println("Limits:")
	fmt.Println("-------")

	if len(limits) == 0 {
		fmt.Println("  (none)")
		fmt.Println()
		return
	}

	for _, lim := range limits {
		fmt.Printf("  Type: %s\n", lim.Type)
		if lim.Frame != "" {
			fmt.Printf("    Frame: %s\n", lim.Frame)
		}
		if lim.BoxType != "" {
			fmt.Printf("    BoxType: %s\n", lim.BoxType)
		}
		fmt.Printf("    Value: %.2f\n", lim.LimitValue)
		fmt.Println()
	}
}

func displayWarnings(warnings []gll.Warning) {
	fmt.Println("Warnings:")
	fmt.Println("---------")

	if len(warnings) == 0 {
		fmt.Println("  (none)")
		fmt.Println()
		return
	}

	for _, warn := range warnings {
		fmt.Printf("  Type: %s\n", warn.Type)
		if warn.Frame != "" {
			fmt.Printf("    Frame: %s\n", warn.Frame)
		}
		fmt.Printf("    Limit: %.2f\n", warn.LimitValue)
		if warn.Text != "" {
			fmt.Printf("    Message: %s\n", warn.Text)
		}
		fmt.Println()
	}
}

func displayFilterGroups(groups []gll.FilterGroup) {
	fmt.Println("Filter Groups:")
	fmt.Println("--------------")

	if len(groups) == 0 {
		fmt.Println("  (none)")
		fmt.Println()
		return
	}

	for _, grp := range groups {
		fmt.Printf("  %s (Key: %s)\n", grp.Label, grp.Key)
		if grp.IsOverridable {
			fmt.Printf("    Overridable: yes\n")
		}
		if len(grp.Filters) > 0 {
			fmt.Printf("    Filters:\n")
			for _, flt := range grp.Filters {
				fmt.Printf("      - %s (Key: %s)\n", flt.Label, flt.Key)
			}
		}
		fmt.Println()
	}
}
