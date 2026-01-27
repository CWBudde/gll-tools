package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cwbudde/gll-tools/pkg/gll"
	"github.com/spf13/cobra"
)

var (
	presetsJSON   bool
	presetsRaw    bool
	presetsDecode bool
)

var presetsCmd = &cobra.Command{
	Use:   "presets [file.gll]",
	Short: "Display system presets from a GLL file",
	Long: `Display system presets from a GLL file.

Examples:
  gllinfo presets speaker.gll          # Show presets
  gllinfo presets speaker.gll --json   # Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runPresets,
}

func init() {
	rootCmd.AddCommand(presetsCmd)

	presetsCmd.Flags().BoolVar(&presetsJSON, "json", false, "output in JSON format")
	presetsCmd.Flags().BoolVar(&presetsRaw, "raw", false, "include raw config bytes in JSON output")
	presetsCmd.Flags().BoolVar(&presetsDecode, "decode", false, "decode config bytes when available")
}

func runPresets(cmd *cobra.Command, args []string) error {
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

	presets := file.Database.Presets
	decoded := make([]*gll.GenSystemConfig, len(presets))
	if presetsDecode {
		for i, p := range presets {
			if len(p.ConfigRaw) == 0 {
				continue
			}
			cfg, err := gll.DecodeGenSystemConfigRaw(p.ConfigRaw)
			if err != nil {
				return fmt.Errorf("decode preset %q: %w", p.Key, err)
			}
			decoded[i] = cfg
		}
	}

	if presetsJSON {
		out := make([]presetOutput, len(presets))
		for i, p := range presets {
			out[i] = presetOutput{
				Label:      p.Label,
				Key:        p.Key,
				ConfigSize: p.ConfigSize,
			}
			if presetsRaw {
				out[i].ConfigRaw = p.ConfigRaw
			}
			if presetsDecode {
				out[i].ConfigDecoded = decoded[i]
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	displayPresets(presets, decoded)
	return nil
}

type presetOutput struct {
	Label         string               `json:"label"`
	Key           string               `json:"key"`
	ConfigSize    int                  `json:"config_size,omitempty"`
	ConfigRaw     []byte               `json:"config_raw,omitempty"`
	ConfigDecoded *gll.GenSystemConfig `json:"config_decoded,omitempty"`
}

func displayPresets(presets []gll.GenSystemPreset, decoded []*gll.GenSystemConfig) {
	fmt.Println("Presets:")
	fmt.Println("--------")

	if len(presets) == 0 {
		fmt.Println("  (none)")
		fmt.Println()
		return
	}

	for i, p := range presets {
		fmt.Printf("  Label: %s\n", p.Label)
		fmt.Printf("    Key: %s\n", p.Key)
		if p.ConfigSize > 0 {
			fmt.Printf("    Config Size: %d bytes\n", p.ConfigSize)
		}
		if presetsDecode && i < len(decoded) && decoded[i] != nil {
			cfg := decoded[i]
			fmt.Printf("    Grid Angle: %.6f\n", cfg.GridAngle)
			fmt.Printf("    Frame Key: %s\n", cfg.FrameKey)
			fmt.Printf("    Cluster Setup Key: %s\n", cfg.ClusterSetupKey)
			fmt.Printf("    System Type: %d\n", cfg.SystemType)
			fmt.Printf("    Elements: %d\n", len(cfg.Elements))
			for j, elem := range cfg.Elements {
				fmt.Printf("      [%d] BoxType=%s Splay=%.2fdeg Gain=%.2f InputConfig=%s Sources=%d\n",
					j,
					elem.BoxTypeKey,
					elem.SplayAngleDeg(),
					elem.Gain,
					elem.InputConfigKey,
					elem.Sources,
				)
			}
		}
		fmt.Println()
	}
}
