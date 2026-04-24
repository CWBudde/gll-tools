package main

import (
	"bytes"
	"encoding/json"

	"github.com/cwbudde/gll-tools/internal/filters"
	"github.com/cwbudde/gll-tools/pkg/gll"
)

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

func computeArrayResponseData(data []byte, configJSON string) ArrayResponseResult {
	reader := bytes.NewReader(data)
	file, err := gll.Parse(reader)
	if err != nil {
		return ArrayResponseResult{
			Success: false,
			Error:   "failed to parse GLL: " + err.Error(),
		}
	}

	var req ArrayResponseRequest
	if err := json.Unmarshal([]byte(configJSON), &req); err != nil {
		return ArrayResponseResult{
			Success: false,
			Error:   "failed to parse config: " + err.Error(),
		}
	}

	return computeArrayResponseForFile(file, reader, req)
}

func computeArrayResponseForFile(
	file *gll.File,
	reader *bytes.Reader,
	req ArrayResponseRequest,
) ArrayResponseResult {
	config := buildArrayConfig(file, reader, req.Elements)
	if len(config.Elements) == 0 {
		return ArrayResponseResult{
			Success: false,
			Error:   "no valid elements in configuration",
		}
	}

	response := gll.ComputeSystemResponseAt(
		config,
		gll.Vector3D{X: req.Receiver.X, Y: req.Receiver.Y, Z: req.Receiver.Z},
		airPropertiesFromInput(req.AirProps),
		req.AirProps.AirAttenOn,
	)
	if response == nil {
		return ArrayResponseResult{
			Success: false,
			Error:   "computation returned no result",
		}
	}

	freqs := make([]float64, len(response.Level))
	for i := range freqs {
		freqs[i] = response.Definition.GetFrequency(i)
	}

	return ArrayResponseResult{
		Success:     true,
		Frequencies: freqs,
		Level:       response.Level,
		Phase:       response.Phase,
	}
}

func marshalArrayResult(result ArrayResponseResult) string {
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

func computeArrayBalloonData(
	data []byte,
	configJSON string,
	progress func(completed, total int),
) ArrayBalloonResult {
	reader := bytes.NewReader(data)
	file, err := gll.Parse(reader)
	if err != nil {
		return ArrayBalloonResult{
			Success: false,
			Error:   "failed to parse GLL: " + err.Error(),
		}
	}

	var req ArrayBalloonRequest
	if err := json.Unmarshal([]byte(configJSON), &req); err != nil {
		return ArrayBalloonResult{
			Success: false,
			Error:   "failed to parse config: " + err.Error(),
		}
	}

	return computeArrayBalloonForFile(file, reader, req, progress)
}

func computeArrayBalloonForFile(
	file *gll.File,
	reader *bytes.Reader,
	req ArrayBalloonRequest,
	progress func(completed, total int),
) ArrayBalloonResult {
	config := buildArrayConfig(file, reader, req.Elements)
	if len(config.Elements) == 0 {
		return ArrayBalloonResult{
			Success: false,
			Error:   "no valid elements in configuration",
		}
	}

	receivers := make([]gll.Vector3D, len(req.Receivers))
	for i, r := range req.Receivers {
		receivers[i] = gll.Vector3D{X: r.X, Y: r.Y, Z: r.Z}
	}

	responses := gll.ComputeSystemResponseGridWithProgress(
		config,
		receivers,
		airPropertiesFromInput(req.AirProps),
		req.AirProps.AirAttenOn,
		progress,
	)

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
			if reader != nil {
				if _, err := reader.Seek(0, 0); err == nil {
					_ = gll.LoadBalloonResponses(reader, srcDef.BalloonData)
				}
			}
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
