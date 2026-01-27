package gll

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type goldenSummary struct {
	Header        Header               `json:"header"`
	GenSystem     genSystemSummary     `json:"gen_system"`
	Metadata      Metadata             `json:"metadata"`
	Database      databaseSummary      `json:"database"`
	ResourceTypes map[ResourceType]int `json:"resource_types"`
	ResourceCount int                  `json:"resource_count"`
	Resources     []resourceSummary    `json:"resources,omitempty"`
	Sources       []sourceSummary      `json:"sources,omitempty"`
}

type genSystemSummary struct {
	Label                        string     `json:"label"`
	Version                      float64    `json:"version"`
	Key                          string     `json:"key"`
	Type                         SystemType `json:"type"`
	Company                      string     `json:"company"`
	InfoText                     string     `json:"info_text,omitempty"`
	CopyrightText                string     `json:"copyright_text,omitempty"`
	SupportText                  string     `json:"support_text,omitempty"`
	WebsiteText                  string     `json:"website_text,omitempty"`
	EmailText                    string     `json:"email_text,omitempty"`
	BackgroundColor              int32      `json:"background_color,omitempty"`
	AllowUserDefinedClusterSetup bool       `json:"allow_user_defined_cluster_setup,omitempty"`
	EnableForSubArrays           bool       `json:"enable_for_sub_arrays,omitempty"`
}

type databaseSummary struct {
	DataFiles         int `json:"data_files"`
	BoxTypes          int `json:"box_types"`
	SourceDefinitions int `json:"source_definitions"`
	FilterGroups      int `json:"filter_groups"`
}

type resourceSummary struct {
	Type             ResourceType `json:"type"`
	Name             string       `json:"name,omitempty"`
	Offset           int64        `json:"offset"`
	Size             int64        `json:"size"`
	SHA256           string       `json:"sha256,omitempty"`
	DecompressedSize int64        `json:"decompressed_size,omitempty"`
	DecompressedSHA  string       `json:"decompressed_sha256,omitempty"`
	Error            string       `json:"error,omitempty"`
}

type sourceSummary struct {
	Key        string                  `json:"key"`
	Definition sourceDefinitionSummary `json:"definition"`
}

type sourceDefinitionSummary struct {
	Label                string          `json:"label"`
	NominalBandwidthFrom float64         `json:"nominal_bandwidth_from"`
	NominalBandwidthTo   float64         `json:"nominal_bandwidth_to"`
	DataType             DataType        `json:"data_type"`
	Balloon              *balloonSummary `json:"balloon,omitempty"`
	OnAxis               *tfSummary      `json:"on_axis,omitempty"`
	Impedance            *tfSummary      `json:"impedance,omitempty"`
}

type balloonSummary struct {
	AngularResolution ResolutionDescriptor `json:"angular_resolution"`
	ResponseCount     int32                `json:"response_count"`
	ResponseVersion   int16                `json:"response_version"`
	Responses         []tfSummary          `json:"responses,omitempty"`
	LoadError         string               `json:"load_error,omitempty"`
}

type tfSummary struct {
	BandsPerOctave int32   `json:"bands_per_octave"`
	StartFreq      float64 `json:"start_freq"`
	PointCount     int32   `json:"point_count"`
	LevelCount     int     `json:"level_count"`
	PhaseCount     int     `json:"phase_count"`
	LevelMin       float64 `json:"level_min"`
	LevelMax       float64 `json:"level_max"`
	PhaseMin       float64 `json:"phase_min"`
	PhaseMax       float64 `json:"phase_max"`
	Delay          float64 `json:"delay"`
	LevelSHA256    string  `json:"level_sha256,omitempty"`
	PhaseSHA256    string  `json:"phase_sha256,omitempty"`
}

func summarizeFile(file *File, r io.ReadSeeker) goldenSummary {
	summary := goldenSummary{
		Header:        file.Header,
		GenSystem:     summarizeGenSystem(file.GenSystem),
		Metadata:      file.Metadata,
		ResourceTypes: make(map[ResourceType]int),
		ResourceCount: len(file.Resources),
	}

	if file.Database != nil {
		summary.Database = databaseSummary{
			DataFiles:         len(file.Database.DataFiles),
			BoxTypes:          len(file.Database.BoxTypes),
			SourceDefinitions: len(file.Database.SourceDefinitions),
			FilterGroups:      len(file.Database.FilterGroups),
		}
	}

	for _, res := range file.Resources {
		summary.ResourceTypes[res.Type]++
	}

	if r != nil {
		summary.Resources = summarizeResources(file, r)
	}

	summary.Sources = summarizeSources(file, r)
	sort.Slice(summary.Sources, func(i, j int) bool {
		return summary.Sources[i].Key < summary.Sources[j].Key
	})

	return summary
}

func summarizeGenSystem(sys GenSystem) genSystemSummary {
	return genSystemSummary{
		Label:                        sys.Label,
		Version:                      sys.Version,
		Key:                          sys.Key,
		Type:                         sys.Type,
		Company:                      sys.Company,
		InfoText:                     sys.InfoText,
		CopyrightText:                sys.CopyrightText,
		SupportText:                  sys.SupportText,
		WebsiteText:                  sys.WebsiteText,
		EmailText:                    sys.EmailText,
		BackgroundColor:              sys.BackgroundColor,
		AllowUserDefinedClusterSetup: sys.AllowUserDefinedClusterSetup,
		EnableForSubArrays:           sys.EnableForSubArrays,
	}
}

func summarizeResources(file *File, r io.ReadSeeker) []resourceSummary {
	if len(file.Resources) == 0 {
		return nil
	}

	resources := make([]resourceSummary, 0, len(file.Resources))
	for _, res := range file.Resources {
		item := resourceSummary{
			Type:   res.Type,
			Name:   res.Name,
			Offset: res.Offset,
			Size:   res.Size,
		}

		data, err := ExtractResource(r, res)
		if err != nil {
			item.Error = err.Error()
			resources = append(resources, item)

			continue
		}

		item.SHA256 = hashBytes(data)

		if res.Type == ResourceTypeZlib {
			decompressed, err := DecompressResource(r, res)
			if err != nil {
				item.Error = err.Error()
			} else {
				item.DecompressedSize = int64(len(decompressed))
				item.DecompressedSHA = hashBytes(decompressed)
			}
		}

		resources = append(resources, item)
	}

	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Offset == resources[j].Offset {
			return resources[i].Name < resources[j].Name
		}

		return resources[i].Offset < resources[j].Offset
	})

	return resources
}

func summarizeSources(file *File, r io.ReadSeeker) []sourceSummary {
	if file.Database == nil || len(file.Database.SourceDefinitions) == 0 {
		return nil
	}

	sources := make([]sourceSummary, 0, len(file.Database.SourceDefinitions))
	for _, item := range file.Database.SourceDefinitions {
		if item.Definition == nil {
			continue
		}

		def := item.Definition
		summary := sourceSummary{
			Key: item.Key,
			Definition: sourceDefinitionSummary{
				Label:                def.Label,
				NominalBandwidthFrom: def.NominalBandwidthFrom,
				NominalBandwidthTo:   def.NominalBandwidthTo,
				DataType:             def.DataType,
			},
		}

		if def.OnAxisSpectrum != nil {
			tf := summarizeTransferFunction(*def.OnAxisSpectrum)
			summary.Definition.OnAxis = &tf
		}

		if def.Impedance != nil {
			tf := summarizeTransferFunction(*def.Impedance)
			summary.Definition.Impedance = &tf
		}

		if def.BalloonData != nil {
			balloon := &balloonSummary{
				AngularResolution: def.BalloonData.AngularResolution,
				ResponseCount:     def.BalloonData.ResponseCount,
				ResponseVersion:   def.BalloonData.ResponseVersion,
			}

			if r == nil {
				balloon.LoadError = "reader unavailable"
			} else if err := LoadBalloonResponses(r, def.BalloonData); err != nil {
				balloon.LoadError = err.Error()
			} else {
				balloon.Responses = make([]tfSummary, 0, len(def.BalloonData.Responses))
				for _, tf := range def.BalloonData.Responses {
					balloon.Responses = append(balloon.Responses, summarizeTransferFunction(tf))
				}
			}

			summary.Definition.Balloon = balloon
		}

		sources = append(sources, summary)
	}

	return sources
}

func summarizeTransferFunction(tf TransferFunction) tfSummary {
	summary := tfSummary{
		BandsPerOctave: tf.Definition.BandsPerOctave,
		StartFreq:      tf.Definition.StartFreq,
		PointCount:     tf.Definition.PointCount,
		LevelCount:     len(tf.Level),
		PhaseCount:     len(tf.Phase),
		Delay:          tf.Delay,
		LevelMin:       math.Inf(1),
		LevelMax:       math.Inf(-1),
		PhaseMin:       math.Inf(1),
		PhaseMax:       math.Inf(-1),
	}

	if len(tf.Level) == 0 {
		summary.LevelMin = 0
		summary.LevelMax = 0
	} else {
		for _, v := range tf.Level {
			if v < summary.LevelMin {
				summary.LevelMin = v
			}

			if v > summary.LevelMax {
				summary.LevelMax = v
			}
		}
	}

	if len(tf.Phase) == 0 {
		summary.PhaseMin = 0
		summary.PhaseMax = 0
	} else {
		for _, v := range tf.Phase {
			if v < summary.PhaseMin {
				summary.PhaseMin = v
			}

			if v > summary.PhaseMax {
				summary.PhaseMax = v
			}
		}
	}

	summary.LevelSHA256 = hashFloat64Slice(tf.Level)
	summary.PhaseSHA256 = hashFloat64Slice(tf.Phase)

	return summary
}

func hashFloat64Slice(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	// Stable hash of float values using IEEE-754 bits in little endian order.
	h := sha256.New()

	buf := make([]byte, 8)
	for _, v := range values {
		binary.LittleEndian.PutUint64(buf, math.Float64bits(v))
		_, _ = h.Write(buf)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func hashBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

func TestParseGoldenFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping golden file tests in short mode")
	}

	testdataDir := os.Getenv("GLL_TESTDATA_DIR")
	if testdataDir == "" {
		testdataDir = filepath.Clean(filepath.Join("..", "..", "testdata", "gll"))
	}

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Skipf("testdata directory not available: %v", err)
	}

	var gllFiles []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) == ".gll" {
			gllFiles = append(gllFiles, filepath.Join(testdataDir, entry.Name()))
		}
	}

	if len(gllFiles) == 0 {
		t.Skip("no .gll files found in testdata")
	}

	sort.Strings(gllFiles)

	update := os.Getenv("GLL_UPDATE_GOLDEN") == "1"
	goldenDir := filepath.Join(testdataDir, "golden")

	for _, path := range gllFiles {
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()

			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("open failed: %v", err)
			}
			defer f.Close()

			parsed, err := Parse(f)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			summary := summarizeFile(parsed, f)

			current, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			goldenPath := filepath.Join(goldenDir, filepath.Base(path)+".json")
			if update {
				if err := os.MkdirAll(goldenDir, 0o755); err != nil {
					t.Fatalf("mkdir failed: %v", err)
				}

				if err := os.WriteFile(goldenPath, append(current, '\n'), 0o644); err != nil {
					t.Fatalf("write golden failed: %v", err)
				}

				return
			}

			golden, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Skipf("golden file missing: %s (set GLL_UPDATE_GOLDEN=1 to create)", goldenPath)
			}

			if !bytes.Equal(bytes.TrimSpace(golden), bytes.TrimSpace(current)) {
				t.Fatalf("golden mismatch: %s (set GLL_UPDATE_GOLDEN=1 to update)", goldenPath)
			}
		})
	}
}
