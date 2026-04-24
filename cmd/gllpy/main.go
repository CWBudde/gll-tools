// Package main provides C-callable exports for Python bindings.
//
// Build with: go build -buildmode=c-shared -o libgll.so ./cmd/gllpy
package main

/*
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

// GLL_Result holds the result of a GLL operation.
// data contains the result (JSON string or binary data).
// error contains an error message if the operation failed.
// length contains the length of data for binary results.
typedef struct {
    char* data;
    char* error;
    int64_t length;
} GLL_Result;

// GLL_ByteResult holds binary data result.
typedef struct {
    void* data;
    int64_t length;
    char* error;
} GLL_ByteResult;

// GLL_Float64Array holds a contiguous float64 buffer plus shape metadata.
typedef struct {
    void* data;
    int64_t rows;
    int64_t cols;
    int64_t length;
    char* error;
} GLL_Float64Array;
*/
import "C"

import (
	"encoding/json"
	"math"
	"os"
	"unsafe"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

// Version of the Python bindings
const version = "0.1.0"

// makeError creates a GLL_Result with an error message.
func makeError(err error) C.GLL_Result {
	return C.GLL_Result{
		data:   nil,
		error:  C.CString(err.Error()),
		length: 0,
	}
}

// makeJSONResult creates a GLL_Result with JSON data.
func makeJSONResult(data any) C.GLL_Result {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return makeError(err)
	}
	return C.GLL_Result{
		data:   C.CString(string(jsonBytes)),
		error:  nil,
		length: C.int64_t(len(jsonBytes)),
	}
}

// makeBytesResult creates a GLL_ByteResult with binary data.
func makeBytesResult(data []byte) C.GLL_ByteResult {
	if len(data) == 0 {
		return C.GLL_ByteResult{
			data:   nil,
			length: 0,
			error:  nil,
		}
	}
	// Allocate C memory and copy data
	cData := C.malloc(C.size_t(len(data)))
	C.memcpy(cData, unsafe.Pointer(&data[0]), C.size_t(len(data)))
	return C.GLL_ByteResult{
		data:   cData,
		length: C.int64_t(len(data)),
		error:  nil,
	}
}

// makeBytesError creates a GLL_ByteResult with an error.
func makeBytesError(err error) C.GLL_ByteResult {
	return C.GLL_ByteResult{
		data:   nil,
		length: 0,
		error:  C.CString(err.Error()),
	}
}

// makeFloat64Array creates a GLL_Float64Array with contiguous float64 data.
func makeFloat64Array(data []float64, rows, cols int) C.GLL_Float64Array {
	if len(data) == 0 {
		return C.GLL_Float64Array{
			data:   nil,
			rows:   C.int64_t(rows),
			cols:   C.int64_t(cols),
			length: 0,
			error:  nil,
		}
	}

	size := C.size_t(len(data)) * C.size_t(unsafe.Sizeof(C.double(0)))
	cData := C.malloc(size)
	C.memcpy(cData, unsafe.Pointer(&data[0]), size)

	return C.GLL_Float64Array{
		data:   cData,
		rows:   C.int64_t(rows),
		cols:   C.int64_t(cols),
		length: C.int64_t(len(data)),
		error:  nil,
	}
}

// makeFloat64ArrayError creates a GLL_Float64Array with an error.
func makeFloat64ArrayError(err error) C.GLL_Float64Array {
	return C.GLL_Float64Array{
		data:   nil,
		rows:   0,
		cols:   0,
		length: 0,
		error:  C.CString(err.Error()),
	}
}

// GLL_Version returns the version string of the library.
//
//export GLL_Version
func GLL_Version() *C.char {
	return C.CString(version)
}

// GLL_ParseFile parses a GLL file and returns JSON metadata.
//
//export GLL_ParseFile
func GLL_ParseFile(path *C.char) C.GLL_Result {
	goPath := C.GoString(path)

	f, err := os.Open(goPath)
	if err != nil {
		return makeError(err)
	}
	defer f.Close()

	gllFile, err := gll.Parse(f)
	if err != nil {
		return makeError(err)
	}

	return makeJSONResult(gllFile)
}

// GLL_ParseBytes parses GLL data from memory and returns JSON metadata.
//
//export GLL_ParseBytes
func GLL_ParseBytes(data *C.char, length C.int64_t) C.GLL_Result {
	if data == nil || length <= 0 {
		return makeError(os.ErrInvalid)
	}

	// Copy data from C to Go
	goData := C.GoBytes(unsafe.Pointer(data), C.int(length))

	// Create a bytes.Reader for parsing
	reader := newBytesReadSeeker(goData)

	gllFile, err := gll.Parse(reader)
	if err != nil {
		return makeError(err)
	}

	return makeJSONResult(gllFile)
}

// GLL_ExtractResource extracts a resource by index from a GLL file.
//
//export GLL_ExtractResource
func GLL_ExtractResource(path *C.char, resourceIndex C.int32_t) C.GLL_ByteResult {
	goPath := C.GoString(path)

	f, err := os.Open(goPath)
	if err != nil {
		return makeBytesError(err)
	}
	defer f.Close()

	gllFile, err := gll.Parse(f)
	if err != nil {
		return makeBytesError(err)
	}

	idx := int(resourceIndex)
	if idx < 0 || idx >= len(gllFile.Resources) {
		return makeBytesError(os.ErrNotExist)
	}

	res := gllFile.Resources[idx]

	// Use DecompressResource for zlib, ExtractResource for others
	var data []byte
	if res.Type == gll.ResourceTypeZlib {
		data, err = gll.DecompressResource(f, res)
	} else {
		data, err = gll.ExtractResource(f, res)
	}

	if err != nil {
		return makeBytesError(err)
	}

	return makeBytesResult(data)
}

// GLL_ExtractDataFile extracts a DataFile by index from a GLL file.
//
//export GLL_ExtractDataFile
func GLL_ExtractDataFile(path *C.char, dataFileIndex C.int32_t) C.GLL_ByteResult {
	goPath := C.GoString(path)

	f, err := os.Open(goPath)
	if err != nil {
		return makeBytesError(err)
	}
	defer f.Close()

	gllFile, err := gll.Parse(f)
	if err != nil {
		return makeBytesError(err)
	}

	if gllFile.Database == nil {
		return makeBytesError(os.ErrNotExist)
	}

	idx := int(dataFileIndex)
	if idx < 0 || idx >= len(gllFile.Database.DataFiles) {
		return makeBytesError(os.ErrNotExist)
	}

	df := gllFile.Database.DataFiles[idx]
	data, err := gll.ExtractDataFile(f, df)
	if err != nil {
		return makeBytesError(err)
	}

	return makeBytesResult(data)
}

// GLL_ExtractIncludeFile extracts an IncludeFile by index from a GLL file.
//
//export GLL_ExtractIncludeFile
func GLL_ExtractIncludeFile(path *C.char, includeFileIndex C.int32_t) C.GLL_ByteResult {
	goPath := C.GoString(path)

	f, err := os.Open(goPath)
	if err != nil {
		return makeBytesError(err)
	}
	defer f.Close()

	gllFile, err := gll.Parse(f)
	if err != nil {
		return makeBytesError(err)
	}

	if gllFile.Database == nil {
		return makeBytesError(os.ErrNotExist)
	}

	idx := int(includeFileIndex)
	if idx < 0 || idx >= len(gllFile.Database.IncludeFiles) {
		return makeBytesError(os.ErrNotExist)
	}

	inc := gllFile.Database.IncludeFiles[idx]
	data, err := gll.ExtractIncludeFile(f, inc)
	if err != nil {
		return makeBytesError(err)
	}

	return makeBytesResult(data)
}

// ArrayConfigJSON is the JSON input format for array calculations.
type ArrayConfigJSON struct {
	GLLPath   string             `json:"gll_path"`
	Elements  []ArrayElementJSON `json:"elements"`
	Receiver  *Vector3DJSON      `json:"receiver,omitempty"`
	Air       *AirPropertiesJSON `json:"air,omitempty"`
	AirAtten  bool               `json:"air_atten"`
	Frequency *float64           `json:"frequency,omitempty"`
}

// ArrayElementJSON represents an element in JSON format.
type ArrayElementJSON struct {
	BoxType  string        `json:"box_type"`
	Position *Vector3DJSON `json:"position,omitempty"`
	Angles   *Vector3DJSON `json:"angles,omitempty"`
	Gain     float64       `json:"gain"`
	Delay    float64       `json:"delay"`
	Muted    bool          `json:"muted"`
}

// Vector3DJSON represents a 3D vector in JSON.
type Vector3DJSON struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// AirPropertiesJSON represents air properties in JSON.
type AirPropertiesJSON struct {
	Temperature *float64 `json:"temperature"`
	Humidity    *float64 `json:"humidity"`
	Pressure    *float64 `json:"pressure"`
}

// ArrayResponseJSON is the JSON output format for array response.
type ArrayResponseJSON struct {
	TransferFunction     *TransferFunctionJSON  `json:"transfer_function,omitempty"`
	CombinedBalloon      *BalloonOutputJSON     `json:"combined_balloon,omitempty"`
	ElementContributions []TransferFunctionJSON `json:"element_contributions,omitempty"`
	Error                string                 `json:"error,omitempty"`
}

// TransferFunctionJSON represents a transfer function in JSON.
type TransferFunctionJSON struct {
	Definition *TransferFunctionDefinitionJSON `json:"definition,omitempty"`
	Level      []float64                       `json:"level"`
	Phase      []float64                       `json:"phase"`
	Delay      float64                         `json:"delay"`
}

// TransferFunctionDefinitionJSON represents the frequency grid for a transfer function.
type TransferFunctionDefinitionJSON struct {
	StartFrequency float64 `json:"start_frequency"`
	EndFrequency   float64 `json:"end_frequency"`
	BandsPerOctave int     `json:"bands_per_octave"`
}

// BalloonOutputJSON represents a single-frequency directivity grid.
type BalloonOutputJSON struct {
	Frequency float64     `json:"frequency"`
	MerStep   float64     `json:"meridian_step"`
	ParStep   float64     `json:"parallel_step"`
	Symmetry  int         `json:"symmetry"`
	Data      [][]float64 `json:"data"`
}

// GLL_ComputeArrayResponse computes the combined array response.
// Input: JSON string with array configuration
// Output: JSON string with transfer function result
//
//export GLL_ComputeArrayResponse
func GLL_ComputeArrayResponse(configJSON *C.char) C.GLL_Result {
	goJSON := C.GoString(configJSON)

	var config ArrayConfigJSON
	if err := json.Unmarshal([]byte(goJSON), &config); err != nil {
		return makeError(err)
	}

	// Open the GLL file
	f, err := os.Open(config.GLLPath)
	if err != nil {
		return makeError(err)
	}
	defer f.Close()

	gllFile, err := gll.Parse(f)
	if err != nil {
		return makeError(err)
	}

	if gllFile.Database == nil {
		return makeJSONResult(ArrayResponseJSON{Error: "no database in GLL file"})
	}

	// Build source definition lookup
	sourceDefMap := make(map[string]*gll.SourceDefinition)
	for i := range gllFile.Database.SourceDefinitions {
		sd := &gllFile.Database.SourceDefinitions[i]
		if sd.Definition != nil {
			if sd.Definition.BalloonData != nil && len(sd.Definition.BalloonData.Responses) == 0 {
				if err := gll.LoadBalloonResponses(f, sd.Definition.BalloonData); err != nil {
					return makeError(err)
				}
			}
			sourceDefMap[sd.Key] = sd.Definition
		}
	}

	// Build box type lookup
	boxTypeMap := make(map[string]*gll.BoxType)
	for i := range gllFile.Database.BoxTypes {
		bt := &gllFile.Database.BoxTypes[i]
		boxTypeMap[bt.Label] = bt
		boxTypeMap[bt.Key] = bt
	}

	// Build array config
	arrayConfig := &gll.ArrayConfig{
		Elements: make([]gll.ArrayElement, 0, len(config.Elements)),
	}

	for _, elem := range config.Elements {
		if elem.Muted {
			continue
		}

		bt, ok := boxTypeMap[elem.BoxType]
		if !ok {
			continue
		}

		arrayElem := gll.ArrayElement{
			Gain: elem.Gain,
		}

		if elem.Position != nil {
			arrayElem.Position = gll.Vector3D{
				X: elem.Position.X,
				Y: elem.Position.Y,
				Z: elem.Position.Z,
			}
		}

		if elem.Angles != nil {
			arrayElem.Angles = gll.Vector3D{
				X: elem.Angles.X,
				Y: elem.Angles.Y,
				Z: elem.Angles.Z,
			}
		}

		// Collect source definitions for this box type
		for _, placement := range bt.SourcePlacements {
			if sd, ok := sourceDefMap[placement.SourceDefKey]; ok {
				arrayElem.SourceDefs = append(arrayElem.SourceDefs, sd)
			}
		}

		if len(arrayElem.SourceDefs) > 0 {
			arrayConfig.Elements = append(arrayConfig.Elements, arrayElem)
		}
	}

	if len(arrayConfig.Elements) == 0 {
		return makeJSONResult(ArrayResponseJSON{Error: "no valid elements in array config"})
	}

	// Set up receiver position (default: 10m on-axis)
	receiver := gll.Vector3D{X: 0, Y: 10, Z: 0}
	if config.Receiver != nil {
		receiver = gll.Vector3D{
			X: config.Receiver.X,
			Y: config.Receiver.Y,
			Z: config.Receiver.Z,
		}
	}

	// Set up air properties
	airProps := gll.DefaultAirProperties()
	if config.Air != nil {
		if config.Air.Temperature != nil {
			airProps.Temperature = *config.Air.Temperature
		}
		if config.Air.Humidity != nil {
			airProps.Humidity = *config.Air.Humidity
		}
		if config.Air.Pressure != nil {
			airProps.Pressure = *config.Air.Pressure
		}
	}

	// Compute response
	details := gll.ComputeSystemResponseDetailsAt(arrayConfig, receiver, airProps, config.AirAtten)
	if details == nil || details.TransferFunction == nil {
		return makeJSONResult(ArrayResponseJSON{Error: "failed to compute response"})
	}

	result := ArrayResponseJSON{
		TransferFunction:     makeTransferFunctionJSON(details.TransferFunction),
		ElementContributions: make([]TransferFunctionJSON, 0, len(details.ElementContributions)),
	}

	for _, contribution := range details.ElementContributions {
		if contributionJSON := makeTransferFunctionJSON(contribution); contributionJSON != nil {
			result.ElementContributions = append(result.ElementContributions, *contributionJSON)
		}
	}

	if config.Frequency != nil {
		result.CombinedBalloon = computeCombinedBalloonJSON(
			arrayConfig,
			receiver,
			airProps,
			config.AirAtten,
			*config.Frequency,
		)
	}

	return makeJSONResult(result)
}

func makeTransferFunctionJSON(tf *gll.TransferFunction) *TransferFunctionJSON {
	if tf == nil {
		return nil
	}
	return &TransferFunctionJSON{
		Definition: &TransferFunctionDefinitionJSON{
			StartFrequency: tf.Definition.StartFreq,
			EndFrequency:   tf.Definition.GetEndFreq(),
			BandsPerOctave: int(tf.Definition.BandsPerOctave),
		},
		Level: tf.Level,
		Phase: tf.Phase,
		Delay: tf.Delay,
	}
}

func computeCombinedBalloonJSON(
	config *gll.ArrayConfig,
	referenceReceiver gll.Vector3D,
	airProps gll.AirProperties,
	airAttenOn bool,
	frequencyHz float64,
) *BalloonOutputJSON {
	const (
		merStep = 10.0
		parStep = 10.0
	)

	radius := math.Sqrt(
		referenceReceiver.X*referenceReceiver.X +
			referenceReceiver.Y*referenceReceiver.Y +
			referenceReceiver.Z*referenceReceiver.Z,
	)
	if radius < 0.01 {
		radius = 10.0
	}

	merCount := int(360.0 / merStep)
	parCount := int(180.0/parStep) + 1
	data := make([][]float64, merCount)

	for m := 0; m < merCount; m++ {
		azimuthDeg := float64(m) * merStep
		data[m] = make([]float64, parCount)
		for p := 0; p < parCount; p++ {
			elevationDeg := -90.0 + float64(p)*parStep
			receiver := receiverOnSphere(radius, azimuthDeg, elevationDeg)
			response := gll.ComputeSystemResponseAt(config, receiver, airProps, airAttenOn)
			data[m][p] = levelAtFrequency(response, frequencyHz)
		}
	}

	return &BalloonOutputJSON{
		Frequency: frequencyHz,
		MerStep:   merStep,
		ParStep:   parStep,
		Symmetry:  int(gll.SymmetryNone),
		Data:      data,
	}
}

func receiverOnSphere(radius, azimuthDeg, elevationDeg float64) gll.Vector3D {
	azimuth := azimuthDeg * math.Pi / 180.0
	elevation := elevationDeg * math.Pi / 180.0
	horizontal := radius * math.Cos(elevation)

	return gll.Vector3D{
		X: horizontal * math.Sin(azimuth),
		Y: horizontal * math.Cos(azimuth),
		Z: radius * math.Sin(elevation),
	}
}

func levelAtFrequency(tf *gll.TransferFunction, frequencyHz float64) float64 {
	if tf == nil || len(tf.Level) == 0 {
		return 0
	}

	bandIdx := 0
	for i := range tf.Level {
		if tf.Definition.GetFrequency(i) >= frequencyHz {
			bandIdx = i
			break
		}
		bandIdx = i
	}

	return tf.Level[bandIdx]
}

// GLL_GetBalloonAtFrequency gets balloon data at a specific frequency.
// Returns JSON with SPL values at each angle.
//
//export GLL_GetBalloonAtFrequency
func GLL_GetBalloonAtFrequency(path *C.char, sourceIndex C.int32_t, frequencyHz C.double) C.GLL_Result {
	goPath := C.GoString(path)

	f, err := os.Open(goPath)
	if err != nil {
		return makeError(err)
	}
	defer f.Close()

	gllFile, err := gll.Parse(f)
	if err != nil {
		return makeError(err)
	}

	if gllFile.Database == nil {
		return makeError(os.ErrNotExist)
	}

	idx := int(sourceIndex)
	if idx < 0 || idx >= len(gllFile.Database.SourceDefinitions) {
		return makeError(os.ErrNotExist)
	}

	sd := gllFile.Database.SourceDefinitions[idx]
	if sd.Definition == nil || sd.Definition.BalloonData == nil {
		return makeError(os.ErrNotExist)
	}

	bd := sd.Definition.BalloonData

	// Find the closest frequency index
	freq := float64(frequencyHz)
	responses := bd.Responses
	if len(responses) == 0 {
		return makeError(os.ErrNotExist)
	}

	// Build balloon grid output
	type BalloonOutput struct {
		Frequency float64     `json:"frequency"`
		MerStep   float64     `json:"meridian_step"`
		ParStep   float64     `json:"parallel_step"`
		Symmetry  int         `json:"symmetry"`
		Data      [][]float64 `json:"data"` // [meridian][parallel]
	}

	merStep := bd.AngularResolution.MeridianStep
	parStep := bd.AngularResolution.ParallelStep
	symmetry := int(bd.AngularResolution.Symmetry)

	// Calculate grid dimensions
	merCount := int(360.0 / merStep)
	parCount := int(180.0/parStep) + 1

	// Find frequency band index
	bandIdx := 0
	if len(responses) > 0 && len(responses[0].Level) > 0 {
		def := responses[0].Definition
		for i := 0; i < len(responses[0].Level); i++ {
			f := def.GetFrequency(i)
			if f >= freq {
				bandIdx = i
				break
			}
			bandIdx = i
		}
	}

	// Build the balloon grid
	data := make([][]float64, merCount)
	for m := 0; m < merCount; m++ {
		data[m] = make([]float64, parCount)
		for p := 0; p < parCount; p++ {
			// Get response at this angle
			phi := float64(m) * merStep      // azimuth
			theta := float64(p)*parStep - 90 // elevation (-90 to 90)

			tf := bd.GetResponseAtAngle(theta*3.14159/180, phi*3.14159/180)
			if tf != nil && bandIdx < len(tf.Level) {
				data[m][p] = tf.Level[bandIdx]
			}
		}
	}

	output := BalloonOutput{
		Frequency: freq,
		MerStep:   merStep,
		ParStep:   parStep,
		Symmetry:  symmetry,
		Data:      data,
	}

	return makeJSONResult(output)
}

// GLL_GetBalloonGridRaw gets a contiguous float64 grid for NumPy zero-copy views.
// The output shape is [meridian][parallel] in row-major order.
//
//export GLL_GetBalloonGridRaw
func GLL_GetBalloonGridRaw(
	path *C.char,
	sourceIndex C.int32_t,
	frequencyHz C.double,
) C.GLL_Float64Array {
	goPath := C.GoString(path)

	f, err := os.Open(goPath)
	if err != nil {
		return makeFloat64ArrayError(err)
	}
	defer f.Close()

	gllFile, err := gll.Parse(f)
	if err != nil {
		return makeFloat64ArrayError(err)
	}

	if gllFile.Database == nil {
		return makeFloat64ArrayError(os.ErrNotExist)
	}

	idx := int(sourceIndex)
	if idx < 0 || idx >= len(gllFile.Database.SourceDefinitions) {
		return makeFloat64ArrayError(os.ErrNotExist)
	}

	sd := gllFile.Database.SourceDefinitions[idx]
	if sd.Definition == nil || sd.Definition.BalloonData == nil {
		return makeFloat64ArrayError(os.ErrNotExist)
	}

	bd := sd.Definition.BalloonData
	if len(bd.Responses) == 0 {
		return makeFloat64ArrayError(os.ErrNotExist)
	}

	merStep := bd.AngularResolution.MeridianStep
	parStep := bd.AngularResolution.ParallelStep
	if merStep <= 0 || parStep <= 0 {
		return makeFloat64ArrayError(os.ErrInvalid)
	}

	merCount := int(360.0 / merStep)
	parCount := int(180.0/parStep) + 1

	freq := float64(frequencyHz)
	bandIdx := 0
	if len(bd.Responses[0].Level) > 0 {
		def := bd.Responses[0].Definition
		for i := 0; i < len(bd.Responses[0].Level); i++ {
			band := def.GetFrequency(i)
			if band >= freq {
				bandIdx = i
				break
			}
			bandIdx = i
		}
	}

	data := make([]float64, merCount*parCount)
	for m := 0; m < merCount; m++ {
		for p := 0; p < parCount; p++ {
			phi := float64(m) * merStep
			theta := float64(p)*parStep - 90

			tf := bd.GetResponseAtAngle(theta*3.14159/180, phi*3.14159/180)
			if tf != nil && bandIdx < len(tf.Level) {
				data[m*parCount+p] = tf.Level[bandIdx]
			}
		}
	}

	return makeFloat64Array(data, merCount, parCount)
}

// GLL_FreeResult frees a GLL_Result's allocated memory.
//
//export GLL_FreeResult
func GLL_FreeResult(result C.GLL_Result) {
	if result.data != nil {
		C.free(unsafe.Pointer(result.data))
	}
	if result.error != nil {
		C.free(unsafe.Pointer(result.error))
	}
}

// GLL_FreeByteResult frees a GLL_ByteResult's allocated memory.
//
//export GLL_FreeByteResult
func GLL_FreeByteResult(result C.GLL_ByteResult) {
	if result.data != nil {
		C.free(result.data)
	}
	if result.error != nil {
		C.free(unsafe.Pointer(result.error))
	}
}

// GLL_FreeFloat64Array frees a GLL_Float64Array's allocated memory.
//
//export GLL_FreeFloat64Array
func GLL_FreeFloat64Array(result C.GLL_Float64Array) {
	if result.data != nil {
		C.free(result.data)
	}
	if result.error != nil {
		C.free(unsafe.Pointer(result.error))
	}
}

// GLL_FreeString frees a C string allocated by this library.
//
//export GLL_FreeString
func GLL_FreeString(s *C.char) {
	if s != nil {
		C.free(unsafe.Pointer(s))
	}
}

// bytesReadSeeker implements io.ReadSeeker for a byte slice.
type bytesReadSeeker struct {
	data   []byte
	offset int64
}

func newBytesReadSeeker(data []byte) *bytesReadSeeker {
	return &bytesReadSeeker{data: data}
}

func (b *bytesReadSeeker) Read(p []byte) (int, error) {
	if b.offset >= int64(len(b.data)) {
		return 0, os.ErrInvalid
	}
	n := copy(p, b.data[b.offset:])
	b.offset += int64(n)
	return n, nil
}

func (b *bytesReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case 0: // io.SeekStart
		newOffset = offset
	case 1: // io.SeekCurrent
		newOffset = b.offset + offset
	case 2: // io.SeekEnd
		newOffset = int64(len(b.data)) + offset
	default:
		return 0, os.ErrInvalid
	}
	if newOffset < 0 {
		return 0, os.ErrInvalid
	}
	b.offset = newOffset
	return newOffset, nil
}

func main() {} // Required but unused for c-shared build mode
