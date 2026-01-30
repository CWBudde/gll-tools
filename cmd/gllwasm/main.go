//go:build js && wasm

// Package main provides a WebAssembly entry point for parsing GLL files in the browser.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"syscall/js"

	"github.com/cwbudde/gll-tools/internal/filters"
	"github.com/cwbudde/gll-tools/internal/mime"
	"github.com/cwbudde/gll-tools/pkg/gll"
)

// WASMResult is the JSON structure returned to JavaScript.
type WASMResult struct {
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
	Data    *WASMData `json:"data,omitempty"`
}

// WASMData contains the parsed GLL file data.
type WASMData struct {
	Header    gll.Header     `json:"header"`
	GenSystem gll.GenSystem  `json:"gen_system"`
	Metadata  gll.Metadata   `json:"metadata"`
	Database  *WASMDatabase  `json:"database,omitempty"`
	Resources []WASMResource `json:"resources,omitempty"`
}

// WASMResource is a web-friendly resource with optional inline image data.
type WASMResource struct {
	Type             gll.ResourceType `json:"type"`
	Name             string           `json:"name,omitempty"`
	Offset           int64            `json:"offset"`
	Size             int64            `json:"size"`
	DecompressedSize int64            `json:"decompressed_size,omitempty"`
	DataURI          string           `json:"data_uri,omitempty"`
}

// WASMDatabase is a simplified database for web display.
type WASMDatabase struct {
	DataFiles         []WASMDataFile         `json:"data_files,omitempty"`
	IncludeFiles      []WASMIncludeFile      `json:"include_files,omitempty"`
	BoxTypes          []gll.BoxType          `json:"box_types,omitempty"`
	Frames            []gll.Frame            `json:"frames,omitempty"`
	Limits            []gll.Limit            `json:"limits,omitempty"`
	Warnings          []gll.Warning          `json:"warnings,omitempty"`
	FilterGroups      []gll.FilterGroup      `json:"filter_groups,omitempty"`
	ClusterSetups     []gll.ClusterSetupItem `json:"cluster_setups,omitempty"`
	Connectors        []gll.Connector        `json:"connectors,omitempty"`
	SourceDefinitions []WASMSourceDefinition `json:"source_definitions,omitempty"`
	Transformers      []gll.Transformer      `json:"transformers,omitempty"`
}

// WASMDataFile is a data file with optional inline data for download.
type WASMDataFile struct {
	Key      string `json:"key"`
	Filename string `json:"filename"`
	Size     int32  `json:"size"`
	DataURI  string `json:"data_uri,omitempty"`
}

// WASMIncludeFile is an include file (PDF, etc.) with inline data for download.
type WASMIncludeFile struct {
	Label    string `json:"label"`
	Key      string `json:"key"`
	Filename string `json:"filename"`
	Size     int32  `json:"size"`
	DataURI  string `json:"data_uri,omitempty"`
}

// WASMSourceDefinition is a source definition with loaded responses.
type WASMSourceDefinition struct {
	Key        string                 `json:"key"`
	Definition *gll.SourceDefinition  `json:"definition"`
	Responses  []WASMTransferFunction `json:"responses,omitempty"`
}

// WASMTransferFunction is a simplified transfer function for display.
type WASMTransferFunction struct {
	Frequencies []float64 `json:"frequencies"`
	Level       []float64 `json:"level"`
	Phase       []float64 `json:"phase"`
	Delay       float64   `json:"delay"`
}

func main() {
	// Register the parseGLL function
	js.Global().Set("parseGLL", js.FuncOf(parseGLL))

	// Register the computeArrayResponse function
	js.Global().Set("computeArrayResponse", js.FuncOf(computeArrayResponse))

	// Register the computeFilterResponse function
	js.Global().Set("computeFilterResponse", js.FuncOf(computeFilterResponse))

	// Register the computeArrayBalloon function
	js.Global().Set("computeArrayBalloon", js.FuncOf(computeArrayBalloon))

	// Keep the program running
	select {}
}

// parseGLL parses a GLL file from a Uint8Array and returns JSON.
func parseGLL(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("no input data provided")
	}

	// Get the Uint8Array from JavaScript
	jsArray := args[0]
	length := jsArray.Get("length").Int()

	// Copy data from JavaScript
	data := make([]byte, length)
	js.CopyBytesToGo(data, jsArray)

	// Parse the GLL file
	reader := bytes.NewReader(data)
	file, err := gll.Parse(reader)
	if err != nil {
		return errorResult("failed to parse GLL: " + err.Error())
	}

	// Convert to WASM-friendly format
	wasmData := &WASMData{
		Header:    file.Header,
		GenSystem: file.GenSystem,
		Metadata:  file.Metadata,
		Resources: buildWASMResources(data, file.Resources),
	}

	// Convert database if present
	if file.Database != nil {
		wasmData.Database = &WASMDatabase{
			DataFiles:     buildWASMDataFiles(data, file.Database.DataFiles),
			IncludeFiles:  buildWASMIncludeFiles(data, file.Database.IncludeFiles),
			BoxTypes:      file.Database.BoxTypes,
			Frames:        file.Database.Frames,
			Limits:        file.Database.Limits,
			Warnings:      file.Database.Warnings,
			FilterGroups:  file.Database.FilterGroups,
			ClusterSetups: file.Database.ClusterSetups,
			Connectors:    file.Database.Connectors,
			Transformers:  file.Database.Transformers,
		}

		// Convert source definitions and load responses
		for _, src := range file.Database.SourceDefinitions {
			wasmSrc := WASMSourceDefinition{
				Key:        src.Key,
				Definition: src.Definition,
			}

			// Load balloon responses if available
			if src.Definition != nil && src.Definition.BalloonData != nil {
				balloon := src.Definition.BalloonData
				if balloon.ResponseCount > 0 && balloon.ResponsesOffset > 0 {
					// Reset reader and load responses
					reader.Seek(0, 0)
					err := gll.LoadBalloonResponses(reader, balloon)
					if err == nil {
						// Convert to WASM format with frequencies
						wasmSrc.Responses = make([]WASMTransferFunction, len(balloon.Responses))
						for i, resp := range balloon.Responses {
							// Generate frequency array
							freqs := make([]float64, len(resp.Level))
							for j := range freqs {
								freqs[j] = resp.Definition.GetFrequency(j)
							}
							wasmSrc.Responses[i] = WASMTransferFunction{
								Frequencies: freqs,
								Level:       resp.Level,
								Phase:       resp.Phase,
								Delay:       resp.Delay,
							}
						}
					}
				}
			}

			wasmData.Database.SourceDefinitions = append(wasmData.Database.SourceDefinitions, wasmSrc)
		}
	}

	// Marshal to JSON
	result := WASMResult{
		Success: true,
		Data:    wasmData,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return errorResult("failed to marshal result: " + err.Error())
	}

	return string(jsonBytes)
}

func errorResult(msg string) string {
	result := WASMResult{
		Success: false,
		Error:   msg,
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}

func buildWASMResources(data []byte, resources []gll.Resource) []WASMResource {
	if len(resources) == 0 {
		return nil
	}

	out := make([]WASMResource, 0, len(resources))
	for _, res := range resources {
		item := WASMResource{
			Type:             res.Type,
			Name:             res.Name,
			Offset:           res.Offset,
			Size:             res.Size,
			DecompressedSize: res.DecompressedSize,
		}

		if res.Type == gll.ResourceTypePNG && res.Offset >= 0 && res.Size > 0 {
			start := int(res.Offset)
			end := start + int(res.Size)
			if start >= 0 && end > start && end <= len(data) {
				item.DataURI = "data:image/png;base64," + base64.StdEncoding.EncodeToString(data[start:end])
			}
		}

		out = append(out, item)
	}

	return out
}

func buildWASMDataFiles(data []byte, dataFiles []gll.DataFile) []WASMDataFile {
	if len(dataFiles) == 0 {
		return nil
	}

	out := make([]WASMDataFile, 0, len(dataFiles))
	for _, df := range dataFiles {
		item := WASMDataFile{
			Key:      df.Key,
			Filename: df.Filename,
			Size:     df.Size,
		}

		// Extract file data for download
		if df.Offset >= 0 && df.Size > 0 {
			start := int(df.Offset)
			end := start + int(df.Size)
			if start >= 0 && end > start && end <= len(data) {
				mimeType := mime.GuessMimeType(df.Filename)
				item.DataURI = "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data[start:end])
			}
		}

		out = append(out, item)
	}

	return out
}

func buildWASMIncludeFiles(data []byte, includeFiles []gll.IncludeFile) []WASMIncludeFile {
	if len(includeFiles) == 0 {
		return nil
	}

	out := make([]WASMIncludeFile, 0, len(includeFiles))
	for _, inc := range includeFiles {
		item := WASMIncludeFile{
			Label:    inc.Label,
			Key:      inc.Key,
			Filename: inc.Filename,
			Size:     inc.Size,
		}

		// Extract file data for download
		if inc.Offset >= 0 && inc.Size > 0 {
			start := int(inc.Offset)
			end := start + int(inc.Size)
			if start >= 0 && end > start && end <= len(data) {
				mimeType := mime.GuessMimeType(inc.Filename)
				item.DataURI = "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data[start:end])
			}
		}

		out = append(out, item)
	}

	return out
}

// ArrayResponseRequest is the input for computeArrayResponse.
type ArrayResponseRequest struct {
	Elements []ArrayElementInput `json:"elements"`
	Receiver ReceiverInput       `json:"receiver"`
	AirProps AirPropsInput       `json:"air_props"`
}

// ArrayElementInput describes one array element.
type ArrayElementInput struct {
	SourceKey string       `json:"source_key"` // Key of the source definition
	Position  Vector3Input `json:"position"`   // Position in meters
	Angles    Vector3Input `json:"angles"`     // Rotation angles in radians
	Gain      float64      `json:"gain"`       // Gain in dB
}

// ReceiverInput is the receiver position.
type ReceiverInput struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Vector3Input is a 3D vector.
type Vector3Input struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// AirPropsInput is air properties for the calculation.
type AirPropsInput struct {
	Temperature float64 `json:"temperature"`  // Celsius
	Humidity    float64 `json:"humidity"`     // 0-1
	Speed       float64 `json:"speed"`        // m/s (optional, calculated if 0)
	AirAttenOn  bool    `json:"air_atten_on"` // Enable air absorption
}

// ArrayResponseResult is the output of computeArrayResponse.
type ArrayResponseResult struct {
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	Frequencies []float64 `json:"frequencies,omitempty"`
	Level       []float64 `json:"level,omitempty"`
	Phase       []float64 `json:"phase,omitempty"`
}

// computeArrayResponse calculates the combined response of an array configuration.
// Input: JSON with { gll_data: Uint8Array, config: ArrayResponseRequest }
func computeArrayResponse(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return marshalArrayResult(ArrayResponseResult{
			Success: false,
			Error:   "requires 2 arguments: gll_data (Uint8Array) and config (JSON string)",
		})
	}

	// Get the GLL data
	jsArray := args[0]
	length := jsArray.Get("length").Int()
	data := make([]byte, length)
	js.CopyBytesToGo(data, jsArray)

	// Parse the GLL file
	reader := bytes.NewReader(data)
	file, err := gll.Parse(reader)
	if err != nil {
		return marshalArrayResult(ArrayResponseResult{
			Success: false,
			Error:   "failed to parse GLL: " + err.Error(),
		})
	}

	// Parse the configuration
	configJSON := args[1].String()
	var req ArrayResponseRequest
	if err := json.Unmarshal([]byte(configJSON), &req); err != nil {
		return marshalArrayResult(ArrayResponseResult{
			Success: false,
			Error:   "failed to parse config: " + err.Error(),
		})
	}

	// Build the array configuration
	config := &gll.ArrayConfig{
		Elements: make([]gll.ArrayElement, 0, len(req.Elements)),
	}

	// Load source definitions and build elements
	for _, elemInput := range req.Elements {
		// Find the source definition
		var srcDef *gll.SourceDefinition
		for _, src := range file.Database.SourceDefinitions {
			if src.Key == elemInput.SourceKey {
				srcDef = src.Definition
				break
			}
		}

		if srcDef == nil {
			continue
		}

		// Load balloon responses if needed
		if srcDef.BalloonData != nil && len(srcDef.BalloonData.Responses) == 0 {
			reader.Seek(0, 0)
			_ = gll.LoadBalloonResponses(reader, srcDef.BalloonData)
		}

		elem := gll.ArrayElement{
			Position: gll.Vector3D{
				X: elemInput.Position.X,
				Y: elemInput.Position.Y,
				Z: elemInput.Position.Z,
			},
			Angles: gll.Vector3D{
				X: elemInput.Angles.X,
				Y: elemInput.Angles.Y,
				Z: elemInput.Angles.Z,
			},
			Gain:       elemInput.Gain,
			SourceDefs: []*gll.SourceDefinition{srcDef},
		}

		config.Elements = append(config.Elements, elem)
	}

	if len(config.Elements) == 0 {
		return marshalArrayResult(ArrayResponseResult{
			Success: false,
			Error:   "no valid elements in configuration",
		})
	}

	// Set up air properties
	airProps := gll.AirProperties{
		Temperature: req.AirProps.Temperature,
		Humidity:    req.AirProps.Humidity,
		Speed:       req.AirProps.Speed,
	}
	if airProps.Speed == 0 {
		airProps.Speed = 343.0 // Default speed of sound
	}
	if airProps.Temperature == 0 {
		airProps.Temperature = 20.0
	}

	// Set up receiver position
	receiver := gll.Vector3D{
		X: req.Receiver.X,
		Y: req.Receiver.Y,
		Z: req.Receiver.Z,
	}

	// Compute the array response
	response := gll.ComputeSystemResponseAt(config, receiver, airProps, req.AirProps.AirAttenOn)

	if response == nil {
		return marshalArrayResult(ArrayResponseResult{
			Success: false,
			Error:   "computation returned no result",
		})
	}

	// Build frequency array
	freqs := make([]float64, len(response.Level))
	for i := range freqs {
		freqs[i] = response.Definition.GetFrequency(i)
	}

	return marshalArrayResult(ArrayResponseResult{
		Success:     true,
		Frequencies: freqs,
		Level:       response.Level,
		Phase:       response.Phase,
	})
}

func marshalArrayResult(result ArrayResponseResult) string {
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}

// computeFilterResponse calculates a filter response for a filter definition.
// Input: gll_data (Uint8Array) and config (JSON string).
func computeFilterResponse(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return marshalFilterResult(filters.FilterResponseResult{
			Success: false,
			Error:   "requires 2 arguments: gll_data (Uint8Array) and config (JSON string)",
		})
	}

	jsArray := args[0]
	length := jsArray.Get("length").Int()
	data := make([]byte, length)
	js.CopyBytesToGo(data, jsArray)

	reader := bytes.NewReader(data)
	file, err := gll.Parse(reader)
	if err != nil {
		return marshalFilterResult(filters.FilterResponseResult{
			Success: false,
			Error:   "failed to parse GLL: " + err.Error(),
		})
	}

	configJSON := args[1].String()
	var req filters.FilterResponseRequest
	if err := json.Unmarshal([]byte(configJSON), &req); err != nil {
		return marshalFilterResult(filters.FilterResponseResult{
			Success: false,
			Error:   "failed to parse config: " + err.Error(),
		})
	}

	result := filters.BuildFilterResponse(file, req)
	return marshalFilterResult(result)
}

func marshalFilterResult(result filters.FilterResponseResult) string {
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}

// ArrayBalloonRequest is the input for computeArrayBalloon.
type ArrayBalloonRequest struct {
	Elements  []ArrayElementInput `json:"elements"`
	Receivers []ReceiverInput     `json:"receivers"`
	AirProps  AirPropsInput       `json:"air_props"`
}

// ArrayBalloonResult is the output of computeArrayBalloon.
type ArrayBalloonResult struct {
	Success     bool                      `json:"success"`
	Error       string                    `json:"error,omitempty"`
	Frequencies []float64                 `json:"frequencies,omitempty"`
	Results     []ArrayBalloonPointResult `json:"results,omitempty"`
}

// ArrayBalloonPointResult is one receiver point's response.
type ArrayBalloonPointResult struct {
	Level []float64 `json:"level"`
	Phase []float64 `json:"phase"`
}

// computeArrayBalloon calculates array response at multiple receiver positions in one call.
func computeArrayBalloon(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return marshalBalloonResult(ArrayBalloonResult{
			Success: false,
			Error:   "requires 2 arguments: gll_data (Uint8Array) and config (JSON string)",
		})
	}

	// Get the GLL data
	jsArray := args[0]
	length := jsArray.Get("length").Int()
	data := make([]byte, length)
	js.CopyBytesToGo(data, jsArray)

	// Parse the GLL file once
	reader := bytes.NewReader(data)
	file, err := gll.Parse(reader)
	if err != nil {
		return marshalBalloonResult(ArrayBalloonResult{
			Success: false,
			Error:   "failed to parse GLL: " + err.Error(),
		})
	}

	// Parse the configuration
	configJSON := args[1].String()
	var req ArrayBalloonRequest
	if err := json.Unmarshal([]byte(configJSON), &req); err != nil {
		return marshalBalloonResult(ArrayBalloonResult{
			Success: false,
			Error:   "failed to parse config: " + err.Error(),
		})
	}

	// Build array config (same as computeArrayResponse)
	config := &gll.ArrayConfig{
		Elements: make([]gll.ArrayElement, 0, len(req.Elements)),
	}

	for _, elemInput := range req.Elements {
		var srcDef *gll.SourceDefinition
		for _, src := range file.Database.SourceDefinitions {
			if src.Key == elemInput.SourceKey {
				srcDef = src.Definition
				break
			}
		}
		if srcDef == nil {
			continue
		}

		// Load balloon responses if needed
		if srcDef.BalloonData != nil && len(srcDef.BalloonData.Responses) == 0 {
			reader.Seek(0, 0)
			_ = gll.LoadBalloonResponses(reader, srcDef.BalloonData)
		}

		elem := gll.ArrayElement{
			Position: gll.Vector3D{
				X: elemInput.Position.X,
				Y: elemInput.Position.Y,
				Z: elemInput.Position.Z,
			},
			Angles: gll.Vector3D{
				X: elemInput.Angles.X,
				Y: elemInput.Angles.Y,
				Z: elemInput.Angles.Z,
			},
			Gain:       elemInput.Gain,
			SourceDefs: []*gll.SourceDefinition{srcDef},
		}
		config.Elements = append(config.Elements, elem)
	}

	if len(config.Elements) == 0 {
		return marshalBalloonResult(ArrayBalloonResult{
			Success: false,
			Error:   "no valid elements in configuration",
		})
	}

	// Build receivers
	receivers := make([]gll.Vector3D, len(req.Receivers))
	for i, r := range req.Receivers {
		receivers[i] = gll.Vector3D{X: r.X, Y: r.Y, Z: r.Z}
	}

	// Set up air properties
	airProps := gll.AirProperties{
		Temperature: req.AirProps.Temperature,
		Humidity:    req.AirProps.Humidity,
		Speed:       req.AirProps.Speed,
	}
	if airProps.Speed == 0 {
		airProps.Speed = 343.0
	}
	if airProps.Temperature == 0 {
		airProps.Temperature = 20.0
	}

	// Compute grid
	responses := gll.ComputeSystemResponseGrid(config, receivers, airProps, req.AirProps.AirAttenOn)

	// Build result
	result := ArrayBalloonResult{
		Success: true,
		Results: make([]ArrayBalloonPointResult, len(responses)),
	}

	for i, resp := range responses {
		if resp == nil {
			result.Results[i] = ArrayBalloonPointResult{
				Level: []float64{},
				Phase: []float64{},
			}
			continue
		}

		// Set frequencies from first non-nil response
		if result.Frequencies == nil {
			freqs := make([]float64, len(resp.Level))
			for j := range freqs {
				freqs[j] = resp.Definition.GetFrequency(j)
			}
			result.Frequencies = freqs
		}

		result.Results[i] = ArrayBalloonPointResult{
			Level: resp.Level,
			Phase: resp.Phase,
		}
	}

	return marshalBalloonResult(result)
}

func marshalBalloonResult(result ArrayBalloonResult) string {
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}
