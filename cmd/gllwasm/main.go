//go:build js && wasm

// Package main provides a WebAssembly entry point for parsing GLL files in the browser.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"syscall/js"
	"time"

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

	// Register the async computeArrayBalloon function
	js.Global().Set("computeArrayBalloonAsync", js.FuncOf(computeArrayBalloonAsync))

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
				mimeType := mimetype.GuessMimeType(df.Filename)
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
				mimeType := mimetype.GuessMimeType(inc.Filename)
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
	SourceKey         string       `json:"source_key"`                   // Key of the source definition
	Position          Vector3Input `json:"position"`                     // Position in meters
	Angles            Vector3Input `json:"angles"`                       // Rotation angles in radians
	OrientationMatrix []float64    `json:"orientation_matrix,omitempty"` // Optional world-from-local matrix
	FilterGroupKeys   []string     `json:"filter_group_keys,omitempty"`  // Internal filter groups linked to this source
	Gain              float64      `json:"gain"`                         // Gain in dB
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
	Temperature *float64 `json:"temperature"`  // Celsius
	Humidity    *float64 `json:"humidity"`     // 0-1
	Pressure    *float64 `json:"pressure"`     // kPa
	Speed       float64  `json:"speed"`        // m/s (optional, calculated if 0)
	AirAttenOn  bool     `json:"air_atten_on"` // Enable air absorption
}

func airPropertiesFromInput(input AirPropsInput) gll.AirProperties {
	airProps := gll.DefaultAirProperties()
	if input.Temperature != nil {
		airProps.Temperature = *input.Temperature
	}
	if input.Humidity != nil {
		airProps.Humidity = *input.Humidity
	}
	if input.Pressure != nil {
		airProps.Pressure = *input.Pressure
	}
	if input.Speed != 0 {
		airProps.Speed = input.Speed
	}
	return airProps
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

	config := buildArrayConfig(file, reader, req.Elements)

	if len(config.Elements) == 0 {
		return marshalArrayResult(ArrayResponseResult{
			Success: false,
			Error:   "no valid elements in configuration",
		})
	}

	airProps := airPropertiesFromInput(req.AirProps)

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

// ArrayBalloonProgressEvent is emitted by computeArrayBalloonAsync.
type ArrayBalloonProgressEvent struct {
	Type      string              `json:"type"`
	Success   bool                `json:"success,omitempty"`
	Error     string              `json:"error,omitempty"`
	Completed int                 `json:"completed,omitempty"`
	Total     int                 `json:"total,omitempty"`
	Result    *ArrayBalloonResult `json:"result,omitempty"`
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

	return marshalBalloonResult(computeArrayBalloonData(data, args[1].String(), nil))
}

// computeArrayBalloonAsync calculates array response at multiple receiver
// positions and emits JSON progress/completion events to a callback.
func computeArrayBalloonAsync(_ js.Value, args []js.Value) any {
	if len(args) < 3 || args[2].Type() != js.TypeFunction {
		return marshalBalloonProgressEvent(ArrayBalloonProgressEvent{
			Type:    "complete",
			Success: false,
			Error:   "requires 3 arguments: gll_data (Uint8Array), config (JSON string), callback",
		})
	}

	jsArray := args[0]
	length := jsArray.Get("length").Int()
	data := make([]byte, length)
	js.CopyBytesToGo(data, jsArray)

	configJSON := args[1].String()
	callback := args[2]

	go func() {
		result := computeArrayBalloonData(data, configJSON, func(completed, total int) {
			callback.Invoke(marshalBalloonProgressEvent(ArrayBalloonProgressEvent{
				Type:      "progress",
				Completed: completed,
				Total:     total,
			}))
			time.Sleep(time.Millisecond)
		})
		callback.Invoke(marshalBalloonProgressEvent(ArrayBalloonProgressEvent{
			Type:    "complete",
			Success: result.Success,
			Error:   result.Error,
			Result:  &result,
		}))
	}()

	return marshalBalloonProgressEvent(ArrayBalloonProgressEvent{
		Type:    "started",
		Success: true,
	})
}

func computeArrayBalloonData(
	data []byte,
	configJSON string,
	progress func(completed, total int),
) ArrayBalloonResult {
	// Parse the GLL file once
	reader := bytes.NewReader(data)
	file, err := gll.Parse(reader)
	if err != nil {
		return ArrayBalloonResult{
			Success: false,
			Error:   "failed to parse GLL: " + err.Error(),
		}
	}

	// Parse the configuration
	var req ArrayBalloonRequest
	if err := json.Unmarshal([]byte(configJSON), &req); err != nil {
		return ArrayBalloonResult{
			Success: false,
			Error:   "failed to parse config: " + err.Error(),
		}
	}

	config := buildArrayConfig(file, reader, req.Elements)

	if len(config.Elements) == 0 {
		return ArrayBalloonResult{
			Success: false,
			Error:   "no valid elements in configuration",
		}
	}

	// Build receivers
	receivers := make([]gll.Vector3D, len(req.Receivers))
	for i, r := range req.Receivers {
		receivers[i] = gll.Vector3D{X: r.X, Y: r.Y, Z: r.Z}
	}

	airProps := airPropertiesFromInput(req.AirProps)

	// Compute grid
	responses := gll.ComputeSystemResponseGridWithProgress(config, receivers, airProps, req.AirProps.AirAttenOn, progress)

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

	return result
}

func marshalBalloonResult(result ArrayBalloonResult) string {
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}

func marshalBalloonProgressEvent(event ArrayBalloonProgressEvent) string {
	jsonBytes, _ := json.Marshal(event)
	return string(jsonBytes)
}

func buildArrayConfig(
	file *gll.File,
	reader *bytes.Reader,
	inputs []ArrayElementInput,
) *gll.ArrayConfig {
	config := &gll.ArrayConfig{
		Elements: make([]gll.ArrayElement, 0, len(inputs)),
	}

	if file == nil || file.Database == nil {
		return config
	}

	for _, elemInput := range inputs {
		srcDef := findSourceDefinition(file, elemInput.SourceKey)
		if srcDef == nil {
			continue
		}

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
		if orientation := parseOrientationMatrix(elemInput.OrientationMatrix); orientation != nil {
			elem.Orientation = orientation
		}
		if spectrum := buildFilterSpectrumForSource(file, srcDef, elemInput.FilterGroupKeys); spectrum != nil {
			elem.FilterSpectra = []*gll.TransferFunction{spectrum}
		}

		config.Elements = append(config.Elements, elem)
	}

	return config
}

func findSourceDefinition(file *gll.File, sourceKey string) *gll.SourceDefinition {
	if file == nil || file.Database == nil {
		return nil
	}
	for _, src := range file.Database.SourceDefinitions {
		if src.Key == sourceKey {
			return src.Definition
		}
	}
	return nil
}

func parseOrientationMatrix(values []float64) *[9]float64 {
	if len(values) != 9 {
		return nil
	}
	matrix := &[9]float64{}
	copy(matrix[:], values)
	return matrix
}

func buildFilterSpectrumForSource(
	file *gll.File,
	srcDef *gll.SourceDefinition,
	groupKeys []string,
) *gll.TransferFunction {
	if file == nil || file.Database == nil || srcDef == nil || len(groupKeys) == 0 {
		return nil
	}
	if srcDef.BalloonData == nil || len(srcDef.BalloonData.Responses) == 0 {
		return nil
	}

	base := &srcDef.BalloonData.Responses[0]
	baseFrequencies := make([]float64, len(base.Level))
	for i := range baseFrequencies {
		baseFrequencies[i] = base.Definition.GetFrequency(i)
	}

	var combined *gll.TransferFunction
	seen := make(map[string]struct{}, len(groupKeys))

	for _, groupKey := range groupKeys {
		if groupKey == "" {
			continue
		}
		if _, ok := seen[groupKey]; ok {
			continue
		}
		seen[groupKey] = struct{}{}

		groupIndex := findFilterGroupIndex(file, groupKey)
		if groupIndex < 0 {
			continue
		}

		group := file.Database.FilterGroups[groupIndex]
		for filterIndex := range group.Filters {
			response := filters.BuildFilterResponse(file, filters.FilterResponseRequest{
				GroupIndex:  groupIndex,
				FilterIndex: filterIndex,
			})
			if !response.Success || len(response.Level) == 0 {
				continue
			}
			if !frequenciesClose(baseFrequencies, response.Frequencies) {
				continue
			}

			filterTF := &gll.TransferFunction{
				Definition: base.Definition,
				Level:      append([]float64(nil), response.Level...),
				Phase:      make([]float64, len(base.Level)),
			}
			if len(response.Phase) == len(filterTF.Phase) {
				copy(filterTF.Phase, response.Phase)
			}

			if combined == nil {
				combined = filterTF
			} else {
				combined.Multiply(filterTF)
			}
		}
	}

	return combined
}

func findFilterGroupIndex(file *gll.File, groupKey string) int {
	if file == nil || file.Database == nil {
		return -1
	}
	for i, group := range file.Database.FilterGroups {
		if group.Key == groupKey {
			return i
		}
	}
	return -1
}

func frequenciesClose(a, b []float64) bool {
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	const tol = 1e-3
	for i := range a {
		av := a[i]
		bv := b[i]
		diff := av - bv
		if diff < 0 {
			diff = -diff
		}
		scale := bv
		if scale < 0 {
			scale = -scale
		}
		if scale < 1 {
			scale = 1
		}
		if diff/scale > tol {
			return false
		}
	}
	return true
}
