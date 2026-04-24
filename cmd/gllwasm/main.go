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

	return marshalArrayResult(computeArrayResponseData(data, args[1].String()))
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
