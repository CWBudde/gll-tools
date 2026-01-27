package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/MeKo-Christian/gll-tools/pkg/gll"
	"github.com/spf13/viper"
)

func runInfoCmd(filename string) error {
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

	// Output based on format
	if viper.GetBool("json") {
		return outputJSON(file)
	}

	outputText(file, filename)
	return nil
}

func outputJSON(file *gll.File) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(file)
}

func outputText(file *gll.File, filename string) {
	verbose := viper.GetBool("verbose")

	fmt.Printf("File: %s\n", filename)
	fmt.Printf("Format: %s v%d (sub: %d)\n", file.Header.Magic, file.Header.FormatVersion, file.Header.SubVersion)
	fmt.Println()

	fmt.Println("=== System ===")
	fmt.Printf("Label:   %s\n", file.GenSystem.Label)
	fmt.Printf("Key:     %s\n", file.GenSystem.Key)
	fmt.Printf("Type:    %s\n", file.GenSystem.Type)
	fmt.Printf("Version: %.1f\n", file.GenSystem.Version)
	fmt.Println()

	fmt.Println("=== Metadata ===")

	if file.GenSystem.Company != "" {
		fmt.Printf("Manufacturer: %s\n", file.GenSystem.Company)
	}

	if file.GenSystem.InfoText != "" {
		fmt.Printf("Description:  %s\n", truncate(file.GenSystem.InfoText, 80))
	}

	if file.GenSystem.CopyrightText != "" {
		fmt.Printf("Copyright:    %s\n", file.GenSystem.CopyrightText)
	}

	if file.GenSystem.WebsiteText != "" {
		fmt.Printf("Website:      %s\n", file.GenSystem.WebsiteText)
	}

	if file.GenSystem.EmailText != "" {
		fmt.Printf("Email:        %s\n", file.GenSystem.EmailText)
	}

	if file.GenSystem.SupportText != "" {
		fmt.Printf("Support:      %s\n", file.GenSystem.SupportText)
	}

	if verbose && len(file.GenSystem.InfoText) > 80 {
		fmt.Println()
		fmt.Println("=== Full Description ===")
		fmt.Println(file.GenSystem.InfoText)
	}

	// Show database contents
	if file.Database != nil {
		if len(file.Database.BoxTypes) > 0 {
			fmt.Println()
			fmt.Println("=== Box Types ===")

			for _, box := range file.Database.BoxTypes {
				fmt.Printf("  %s\n", box.Label)
			}
		}

		if verbose && len(file.Database.DataFiles) > 0 {
			fmt.Println()
			fmt.Println("=== Data Files ===")

			for _, df := range file.Database.DataFiles {
				fmt.Printf("  %s (%d bytes)\n", df.Filename, df.Size)
			}
		}

		if len(file.Database.SourceDefinitions) > 0 {
			fmt.Println()
			fmt.Println("=== Source Definitions ===")

			for _, src := range file.Database.SourceDefinitions {
				if src.Definition != nil {
					def := src.Definition
					fmt.Printf("  %s: %.0f-%.0f Hz (%s)\n",
						def.Label,
						def.NominalBandwidthFrom,
						def.NominalBandwidthTo,
						def.DataType)

					// Show balloon details in verbose mode
					if verbose && def.BalloonData != nil {
						balloon := def.BalloonData
						symmetryNames := []string{"None", "Vertical", "Horizontal", "Quarter", "Axial"}

						symName := "Unknown"
						if int(balloon.AngularResolution.Symmetry) < len(symmetryNames) {
							symName = symmetryNames[balloon.AngularResolution.Symmetry]
						}

						fmt.Printf("    Balloon: %s symmetry, %.1f° x %.1f° grid, %d responses\n",
							symName,
							balloon.AngularResolution.MeridianStep,
							balloon.AngularResolution.ParallelStep,
							balloon.ResponseCount)
					}
				} else {
					fmt.Printf("  %s\n", src.Key)
				}
			}
		}
	}

	// Show embedded resources (from resource scan)
	if len(file.Resources) > 0 && !verbose {
		fmt.Println()
		fmt.Println("=== Embedded Resources ===")

		for _, res := range file.Resources {
			fmt.Printf("  %s: %s (%d bytes)\n", res.Type, res.Name, res.Size)
		}
	}

	// Show hash info if verbose
	if verbose {
		fmt.Println()
		fmt.Println("=== Internal ===")
		fmt.Printf("Checksum: %x\n", file.Header.Checksum)
		fmt.Printf("Hash ID:  %x\n", file.Header.HashID)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
