package viz

import (
	"fmt"
	"io"
	"math"
	"strings"
)

type PolarPlot struct {
	Width, Height int
	Title         string
	FrequencyHz   float64
	AnglesDeg     []float64
	Horizontal    []float64
	Vertical      []float64
	Normalize     bool
	UsesOnAxis    bool
}

type ResponsePlotKind string

const (
	ResponseMagnitude    ResponsePlotKind = "magnitude"
	ResponsePhaseWrapped ResponsePlotKind = "phase-wrapped"
	ResponsePhaseUnwrap  ResponsePlotKind = "phase-unwrapped"
	ResponseGroupDelay   ResponsePlotKind = "group-delay"
)

type ResponsePlot struct {
	Width, Height int
	Title         string
	Frequencies   []float64
	Series        []float64
	Kind          ResponsePlotKind
	UsesOnAxis    bool
}

func RenderPolarSVG(w io.Writer, plot PolarPlot) error {
	// Apply default size
	width := plot.Width
	height := plot.Height
	if width <= 0 {
		width = 900
	}
	if height <= 0 {
		height = 700
	}

	// Determine level range for scaling
	levelMax, levelMin := levelRange(plot.Horizontal, plot.Vertical)
	if !isFinite(levelMax) {
		return fmt.Errorf("no valid polar levels")
	}
	if isFinite(levelMin) {
		levelMax += 3
		levelMin = levelMax - 40
	} else {
		levelMin = levelMax - 40
	}

	cx := float64(width) / 2
	cy := float64(height) / 2
	plotRadius := math.Min(float64(width), float64(height)) * 0.35

	// Build title with frequency and flags
	title := plot.Title
	if title == "" {
		title = "Polar Directivity"
	}
	if plot.FrequencyHz > 0 {
		title = fmt.Sprintf("%s @ %s", title, formatHz(plot.FrequencyHz))
	}
	if plot.Normalize {
		title += " (normalized)"
	}
	if plot.UsesOnAxis {
		title += " (on-axis combined)"
	}

	// SVG header + background
	fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	fmt.Fprintf(w, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\">\n", width, height, width, height)
	fmt.Fprintf(w, "<rect width=\"100%%\" height=\"100%%\" fill=\"#0f172a\"/>\n")
	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#e2e8f0\" font-family=\"sans-serif\" font-size=\"18\" text-anchor=\"middle\">%s</text>\n", cx, 32.0, escapeText(title))

	// Grid rings
	// Grid rings
	gridStep := 10.0
	for lvl := levelMax; lvl >= levelMin-0.001; lvl -= gridStep {
		r := scaleRadius(lvl, levelMin, levelMax, plotRadius)
		fmt.Fprintf(w, "<circle cx=\"%.1f\" cy=\"%.1f\" r=\"%.2f\" fill=\"none\" stroke=\"rgba(148,163,184,0.2)\" stroke-width=\"1\"/>\n", cx, cy, r)
		label := fmt.Sprintf("%.0f dB", lvl)
		fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#94a3b8\" font-family=\"sans-serif\" font-size=\"11\" text-anchor=\"middle\">%s</text>\n", cx, cy-r-6, escapeText(label))
	}

	// Angle lines: faint dashed every 10 degrees
	for ang := 0.0; ang < 360; ang += 10 {
		x, y := polarPoint(cx, cy, plotRadius, ang)
		fmt.Fprintf(w, "<line x1=\"%.1f\" y1=\"%.1f\" x2=\"%.1f\" y2=\"%.1f\" stroke=\"rgba(148,163,184,0.18)\" stroke-width=\"1\" stroke-dasharray=\"4 4\"/>\n", cx, cy, x, y)
	}

	// Angle labels every 10 degrees, placed outside circle and away from compass labels
	for ang := 0.0; ang < 360; ang += 10 {
		if isNearCompassLabel(ang) {
			continue
		}
		x, y := polarPoint(cx, cy, plotRadius+18, ang)
		label := fmt.Sprintf("%d°", int(ang))
		fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#94a3b8\" font-family=\"sans-serif\" font-size=\"9\" text-anchor=\"middle\" dominant-baseline=\"middle\">%s</text>\n", x, y, escapeText(label))
	}

	// Labels (compass)
	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#e2e8f0\" font-family=\"sans-serif\" font-size=\"12\" text-anchor=\"start\" dominant-baseline=\"middle\">Front</text>\n", cx+plotRadius+32, cy)
	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#e2e8f0\" font-family=\"sans-serif\" font-size=\"12\" text-anchor=\"end\" dominant-baseline=\"middle\">Back</text>\n", cx-plotRadius-32, cy)

	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#2563eb\" font-family=\"sans-serif\" font-size=\"12\" text-anchor=\"middle\" dominant-baseline=\"baseline\">Right</text>\n", cx-20, cy-plotRadius-24)
	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#dc2626\" font-family=\"sans-serif\" font-size=\"12\" text-anchor=\"middle\" dominant-baseline=\"baseline\">Top</text>\n", cx+20, cy-plotRadius-24)

	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#2563eb\" font-family=\"sans-serif\" font-size=\"12\" text-anchor=\"middle\" dominant-baseline=\"hanging\">Left</text>\n", cx-20, cy+plotRadius+24)
	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#dc2626\" font-family=\"sans-serif\" font-size=\"12\" text-anchor=\"middle\" dominant-baseline=\"hanging\">Bottom</text>\n", cx+20, cy+plotRadius+24)

	// Build dataset paths
	hPath := buildPolarPath(plot.AnglesDeg, plot.Horizontal, levelMin, levelMax, cx, cy, plotRadius)
	vPath := buildPolarPath(plot.AnglesDeg, plot.Vertical, levelMin, levelMax, cx, cy, plotRadius)

	// Draw datasets
	if hPath != "" {
		fmt.Fprintf(w, "<path d=\"%s\" fill=\"rgba(37,99,235,0.15)\" stroke=\"#2563eb\" stroke-width=\"2\"/>\n", hPath)
	}
	if vPath != "" {
		fmt.Fprintf(w, "<path d=\"%s\" fill=\"rgba(220,38,38,0.15)\" stroke=\"#dc2626\" stroke-width=\"2\"/>\n", vPath)
	}

	// Legend
	legendX := 60.0
	legendY := float64(height) - 40
	fmt.Fprintf(w, "<rect x=\"%.1f\" y=\"%.1f\" width=\"16\" height=\"6\" fill=\"#2563eb\"/>\n", legendX, legendY-6)
	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#cbd5f5\" font-family=\"sans-serif\" font-size=\"12\" text-anchor=\"start\">Horizontal</text>\n", legendX+22, legendY)

	legendX += 140
	fmt.Fprintf(w, "<rect x=\"%.1f\" y=\"%.1f\" width=\"16\" height=\"6\" fill=\"#dc2626\"/>\n", legendX, legendY-6)
	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#fecaca\" font-family=\"sans-serif\" font-size=\"12\" text-anchor=\"start\">Vertical</text>\n", legendX+22, legendY)

	fmt.Fprint(w, "</svg>\n")
	return nil
}

func RenderResponseSVG(w io.Writer, plot ResponsePlot) error {
	// Apply default size
	width := plot.Width
	height := plot.Height
	if width <= 0 {
		width = 1000
	}
	if height <= 0 {
		height = 700
	}

	// Validate inputs
	if len(plot.Frequencies) == 0 || len(plot.Series) == 0 {
		return fmt.Errorf("missing response data")
	}

	minFreq, maxFreq := minMax(plot.Frequencies)
	if minFreq <= 0 || maxFreq <= 0 || minFreq == maxFreq {
		return fmt.Errorf("invalid frequency range")
	}

	// Determine value range with padding
	seriesMin, seriesMax := minMax(plot.Series)
	if !isFinite(seriesMin) || !isFinite(seriesMax) {
		return fmt.Errorf("invalid level data")
	}
	pad := 10.0
	if plot.Kind == ResponsePhaseWrapped || plot.Kind == ResponsePhaseUnwrap {
		pad = 1
	}
	if plot.Kind == ResponseGroupDelay {
		pad = 2
	}
	seriesMin, seriesMax = padRange(seriesMin, seriesMax, pad)

	marginLeft := 70.0
	marginRight := 40.0
	marginTop := 50.0
	marginBottom := 60.0
	plotHeight := float64(height) - marginTop - marginBottom
	plotWidth := float64(width) - marginLeft - marginRight

	// Build title text
	title := plot.Title
	if title == "" {
		title = "Frequency Response"
	}
	title = fmt.Sprintf("%s (%s)", title, responseKindLabel(plot.Kind))
	if plot.UsesOnAxis {
		title += " (on-axis combined)"
	}

	// SVG header + background
	fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	fmt.Fprintf(w, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\">\n", width, height, width, height)
	fmt.Fprintf(w, "<rect width=\"100%%\" height=\"100%%\" fill=\"#0b1120\"/>\n")
	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#e2e8f0\" font-family=\"sans-serif\" font-size=\"18\" text-anchor=\"middle\">%s</text>\n", float64(width)/2, 30.0, escapeText(title))

	plotY := marginTop
	// Draw main response panel
	drawResponsePanel(w, responsePanel{
		x:          marginLeft,
		y:          plotY,
		w:          plotWidth,
		h:          plotHeight,
		minFreq:    minFreq,
		maxFreq:    maxFreq,
		minValue:   seriesMin,
		maxValue:   seriesMax,
		series:     plot.Series,
		freqs:      plot.Frequencies,
		label:      responseKindLabel(plot.Kind),
		lineColor:  responseLineColor(plot.Kind),
		axisColor:  "rgba(148,163,184,0.35)",
		textColor:  "#94a3b8",
		tickFormat: responseTickFormat(plot.Kind),
	})

	// X-axis labels at bottom
	ticks := logTicks(minFreq, maxFreq)
	for _, tick := range ticks {
		x := scaleLog(tick, minFreq, maxFreq, marginLeft, marginLeft+plotWidth)
		fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#94a3b8\" font-family=\"sans-serif\" font-size=\"11\" text-anchor=\"middle\">%s</text>\n", x, float64(height)-28, escapeText(formatHzLabel(tick)))
	}
	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"#94a3b8\" font-family=\"sans-serif\" font-size=\"12\" text-anchor=\"middle\">Frequency (Hz)</text>\n", marginLeft+plotWidth/2, float64(height)-10)

	fmt.Fprint(w, "</svg>\n")
	return nil
}

type responsePanel struct {
	x, y       float64
	w, h       float64
	minFreq    float64
	maxFreq    float64
	minValue   float64
	maxValue   float64
	series     []float64
	freqs      []float64
	label      string
	lineColor  string
	axisColor  string
	textColor  string
	tickFormat func(float64) string
}

func drawResponsePanel(w io.Writer, panel responsePanel) {
	// Panel frame and label
	fmt.Fprintf(w, "<rect x=\"%.1f\" y=\"%.1f\" width=\"%.1f\" height=\"%.1f\" fill=\"none\" stroke=\"%s\" stroke-width=\"1\"/>\n", panel.x, panel.y, panel.w, panel.h, panel.axisColor)
	fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"%s\" font-family=\"sans-serif\" font-size=\"12\" text-anchor=\"start\">%s</text>\n", panel.x, panel.y-8, panel.textColor, escapeText(panel.label))

	// Horizontal grid
	// Horizontal grid
	step := gridStep(panel.minValue, panel.maxValue)
	for v := math.Ceil(panel.minValue/step) * step; v <= panel.maxValue+0.0001; v += step {
		y := scaleLinear(v, panel.minValue, panel.maxValue, panel.y+panel.h, panel.y)
		fmt.Fprintf(w, "<line x1=\"%.1f\" y1=\"%.1f\" x2=\"%.1f\" y2=\"%.1f\" stroke=\"%s\" stroke-width=\"1\"/>\n", panel.x, y, panel.x+panel.w, y, panel.axisColor)
		fmt.Fprintf(w, "<text x=\"%.1f\" y=\"%.1f\" fill=\"%s\" font-family=\"sans-serif\" font-size=\"11\" text-anchor=\"end\" dominant-baseline=\"middle\">%s</text>\n", panel.x-8, y, panel.textColor, escapeText(panel.tickFormat(v)))
	}

	// Vertical grid (log frequency)
	for _, tick := range logTicks(panel.minFreq, panel.maxFreq) {
		x := scaleLog(tick, panel.minFreq, panel.maxFreq, panel.x, panel.x+panel.w)
		fmt.Fprintf(w, "<line x1=\"%.1f\" y1=\"%.1f\" x2=\"%.1f\" y2=\"%.1f\" stroke=\"%s\" stroke-width=\"1\"/>\n", x, panel.y, x, panel.y+panel.h, panel.axisColor)
	}

	// Series path
	path := buildResponsePath(panel.freqs, panel.series, panel.minFreq, panel.maxFreq, panel.minValue, panel.maxValue, panel.x, panel.y, panel.w, panel.h)
	if path != "" {
		fmt.Fprintf(w, "<path d=\"%s\" fill=\"none\" stroke=\"%s\" stroke-width=\"2\"/>\n", path, panel.lineColor)
	}
}

func responseKindLabel(kind ResponsePlotKind) string {
	// Human-readable label by series type
	switch kind {
	case ResponsePhaseWrapped:
		return "Phase (rad, wrapped)"
	case ResponsePhaseUnwrap:
		return "Phase (rad, unwrapped)"
	case ResponseGroupDelay:
		return "Group Delay (ms)"
	default:
		return "Level (dB)"
	}
}

func responseLineColor(kind ResponsePlotKind) string {
	// Color palette by series type
	switch kind {
	case ResponsePhaseWrapped, ResponsePhaseUnwrap:
		return "#f97316"
	case ResponseGroupDelay:
		return "#22c55e"
	default:
		return "#38bdf8"
	}
}

func responseTickFormat(kind ResponsePlotKind) func(float64) string {
	// Tick formatting by series type
	switch kind {
	case ResponsePhaseWrapped, ResponsePhaseUnwrap:
		return func(v float64) string { return fmt.Sprintf("%.2f", v) }
	case ResponseGroupDelay:
		return func(v float64) string { return fmt.Sprintf("%.1f", v) }
	default:
		return func(v float64) string { return fmt.Sprintf("%.0f", v) }
	}
}

func buildResponsePath(freqs, series []float64, minFreq, maxFreq, minValue, maxValue, x, y, w, h float64) string {
	// Build SVG path for response series
	if len(freqs) == 0 || len(series) == 0 {
		return ""
	}
	var b strings.Builder
	started := false
	for i := range freqs {
		if i >= len(series) {
			break
		}
		freq := freqs[i]
		val := series[i]
		if freq <= 0 || !isFinite(val) {
			continue
		}
		px := scaleLog(freq, minFreq, maxFreq, x, x+w)
		py := scaleLinear(val, minValue, maxValue, y+h, y)
		if !started {
			fmt.Fprintf(&b, "M %.2f %.2f", px, py)
			started = true
		} else {
			fmt.Fprintf(&b, " L %.2f %.2f", px, py)
		}
	}
	return b.String()
}

func buildPolarPath(angles []float64, levels []float64, minLevel, maxLevel, cx, cy, radius float64) string {
	// Build SVG path for polar dataset
	if len(angles) == 0 || len(levels) == 0 {
		return ""
	}
	var b strings.Builder
	started := false
	for i, ang := range angles {
		if i >= len(levels) {
			break
		}
		level := levels[i]
		if !isFinite(level) {
			continue
		}
		level = clamp(level, minLevel, maxLevel)
		r := scaleRadius(level, minLevel, maxLevel, radius)
		x, y := polarPoint(cx, cy, r, ang)
		if !started {
			fmt.Fprintf(&b, "M %.2f %.2f", x, y)
			started = true
		} else {
			fmt.Fprintf(&b, " L %.2f %.2f", x, y)
		}
	}
	if started {
		b.WriteString(" Z")
	}
	return b.String()
}

func polarPoint(cx, cy, radius, angleDeg float64) (float64, float64) {
	// Convert polar angle to SVG coordinates
	rad := angleDeg * math.Pi / 180.0
	x := cx + radius*math.Cos(rad)
	y := cy - radius*math.Sin(rad)
	return x, y
}

func isNearCompassLabel(angleDeg float64) bool {
	// Avoid overlap with compass labels
	// Avoid overlap with Front/Back (0/180) and Right/Top/Left/Bottom (90/270)
	const threshold = 8.0
	for _, target := range []float64{0, 90, 180, 270} {
		delta := math.Abs(math.Mod(angleDeg-target+180, 360) - 180)
		if delta <= threshold {
			return true
		}
	}
	return false
}

func scaleRadius(value, minValue, maxValue, radius float64) float64 {
	// Scale level to radial distance
	if maxValue == minValue {
		return radius
	}
	return (value - minValue) / (maxValue - minValue) * radius
}

func scaleLinear(value, minValue, maxValue, minPx, maxPx float64) float64 {
	// Linear scale mapping
	if maxValue == minValue {
		return minPx
	}
	return minPx + (value-minValue)/(maxValue-minValue)*(maxPx-minPx)
}

func scaleLog(value, minValue, maxValue, minPx, maxPx float64) float64 {
	// Logarithmic scale mapping
	if value <= 0 || minValue <= 0 || maxValue <= 0 || minValue == maxValue {
		return minPx
	}
	logMin := math.Log10(minValue)
	logMax := math.Log10(maxValue)
	return minPx + (math.Log10(value)-logMin)/(logMax-logMin)*(maxPx-minPx)
}

func logTicks(minFreq, maxFreq float64) []float64 {
	candidates := []float64{10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000, 40000}
	out := make([]float64, 0, len(candidates))
	for _, f := range candidates {
		if f >= minFreq && f <= maxFreq {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		out = append(out, minFreq, maxFreq)
	}
	return out
}

func formatHzLabel(freq float64) string {
	if freq >= 1000 {
		return fmt.Sprintf("%.0fk", freq/1000)
	}
	return fmt.Sprintf("%.0f", freq)
}

func formatHz(freq float64) string {
	if freq >= 1000 {
		return fmt.Sprintf("%.1f kHz", freq/1000)
	}
	return fmt.Sprintf("%.0f Hz", freq)
}

func escapeText(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(s)
}

func levelRange(levelsA, levelsB []float64) (float64, float64) {
	maxVal := math.NaN()
	minVal := math.NaN()
	for _, val := range append(levelsA, levelsB...) {
		if !isFinite(val) {
			continue
		}
		if !isFinite(maxVal) || val > maxVal {
			maxVal = val
		}
		if !isFinite(minVal) || val < minVal {
			minVal = val
		}
	}
	return maxVal, minVal
}

func minMax(values []float64) (float64, float64) {
	minVal := math.NaN()
	maxVal := math.NaN()
	for _, val := range values {
		if !isFinite(val) {
			continue
		}
		if !isFinite(minVal) || val < minVal {
			minVal = val
		}
		if !isFinite(maxVal) || val > maxVal {
			maxVal = val
		}
	}
	return minVal, maxVal
}

func padRange(minVal, maxVal, step float64) (float64, float64) {
	if minVal == maxVal {
		return minVal - step, maxVal + step
	}
	span := maxVal - minVal
	pad := span * 0.1
	if pad < step {
		pad = step
	}
	return minVal - pad, maxVal + pad
}

func gridStep(minVal, maxVal float64) float64 {
	span := math.Abs(maxVal - minVal)
	switch {
	case span > 60:
		return 10
	case span > 30:
		return 5
	case span > 10:
		return 2
	default:
		return 1
	}
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func clamp(value, minVal, maxVal float64) float64 {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}
