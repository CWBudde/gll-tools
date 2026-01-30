package filters

import (
	"math"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

// TestFactorial tests factorial calculation
func TestFactorial(t *testing.T) {
	tests := []struct {
		n    int
		want float64
	}{
		{0, 1.0},
		{1, 1.0},
		{2, 2.0},
		{3, 6.0},
		{4, 24.0},
		{5, 120.0},
		{10, 3628800.0},
	}

	for _, tt := range tests {
		got := factorial(tt.n)
		if got != tt.want {
			t.Errorf("factorial(%d) = %f, want %f", tt.n, got, tt.want)
		}
	}
}

// TestComplexAdd tests complex array addition
func TestComplexAdd(t *testing.T) {
	a := []complex128{complex(1, 2), complex(3, 4)}
	b := []complex128{complex(5, 6), complex(7, 8)}

	result := complexAdd(a, b)

	expected := []complex128{complex(6, 8), complex(10, 12)}
	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("complexAdd[%d] = %v, want %v", i, result[i], expected[i])
		}
	}
}

// TestComplexAddMismatch tests mismatched array lengths
func TestComplexAddMismatch(t *testing.T) {
	a := []complex128{complex(1, 2)}
	b := []complex128{complex(5, 6), complex(7, 8)}

	result := complexAdd(a, b)
	if result != nil {
		t.Error("complexAdd with mismatched lengths should return nil")
	}
}

// TestComplexAddEmpty tests empty arrays
func TestComplexAddEmpty(t *testing.T) {
	result := complexAdd(nil, nil)
	if result != nil {
		t.Error("complexAdd with nil inputs should return nil")
	}
}

// TestComplexDivide tests complex array division
func TestComplexDivide(t *testing.T) {
	a := []complex128{complex(10, 0), complex(100, 0)}
	b := []complex128{complex(2, 0), complex(10, 0)}

	result := complexDivide(a, b)

	expected := []complex128{complex(5, 0), complex(10, 0)}
	for i := range result {
		if cmplxAbs(result[i]-expected[i]) > 1e-10 {
			t.Errorf("complexDivide[%d] = %v, want %v", i, result[i], expected[i])
		}
	}
}

// TestComplexScale tests scaling complex array
func TestComplexScale(t *testing.T) {
	a := []complex128{complex(2, 3), complex(4, 5)}
	scale := 2.0

	result := complexScale(a, scale)

	expected := []complex128{complex(4, 6), complex(8, 10)}
	for i := range result {
		if cmplxAbs(result[i]-expected[i]) > 1e-10 {
			t.Errorf("complexScale[%d] = %v, want %v", i, result[i], expected[i])
		}
	}
}

// TestComplexScaleEmpty tests scaling empty array
func TestComplexScaleEmpty(t *testing.T) {
	result := complexScale(nil, 2.0)
	if result != nil {
		t.Error("complexScale with nil input should return nil")
	}
}

// TestComplexAddScalar tests adding scalar to complex array
func TestComplexAddScalar(t *testing.T) {
	a := []complex128{complex(1, 2), complex(3, 4)}
	scalar := 5.0

	result := complexAddScalar(a, scalar)

	expected := []complex128{complex(6, 2), complex(8, 4)}
	for i := range result {
		if cmplxAbs(result[i]-expected[i]) > 1e-10 {
			t.Errorf("complexAddScalar[%d] = %v, want %v", i, result[i], expected[i])
		}
	}
}

// TestButterworthNormalization tests Butterworth normalization calculation
func TestButterworthNormalization(t *testing.T) {
	tests := []struct {
		name       string
		order      int
		alignScale float64
		wantMin    float64
		wantMax    float64
	}{
		{"Order 1, 3dB", 1, 0.5, 0.9, 1.1},
		{"Order 2, 3dB", 2, 0.5, 0.7, 0.72}, // sqrt(2)/2 ≈ 0.707
		{"Zero order", 0, 0.5, 1.0, 1.0},
		{"Zero alignScale", 2, 0.0, 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := butterworthNormalization(tt.order, tt.alignScale)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("butterworthNormalization(%d, %f) = %f, want between %f and %f",
					tt.order, tt.alignScale, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestButterworthCoefficients tests Butterworth coefficient generation
func TestButterworthCoefficients(t *testing.T) {
	// Test order 2 (basic case)
	coeffs := butterworthCoefficients(2)
	if coeffs == nil || len(coeffs) != 3 {
		t.Errorf("butterworthCoefficients(2) returned wrong length: got %d, want 3", len(coeffs))
	}

	// All coefficients should be positive
	for i, c := range coeffs {
		if c <= 0 {
			t.Errorf("Coefficient[%d] should be positive, got %f", i, c)
		}
	}

	// Test zero order
	coeffs = butterworthCoefficients(0)
	if coeffs != nil {
		t.Error("butterworthCoefficients(0) should return nil")
	}
}

// TestBesselCoefficients tests Bessel coefficient generation
func TestBesselCoefficients(t *testing.T) {
	// Test order 2
	coeffs := besselCoefficients(2)
	if coeffs == nil || len(coeffs) != 3 {
		t.Errorf("besselCoefficients(2) returned wrong length: got %d, want 3", len(coeffs))
	}

	// Test zero order
	coeffs = besselCoefficients(0)
	if coeffs != nil {
		t.Error("besselCoefficients(0) should return nil")
	}
}

// TestBesselCoefficient tests individual Bessel coefficient calculation
func TestBesselCoefficient(t *testing.T) {
	// First coefficient should always be positive
	coeff := besselCoefficient(2, 0)
	if coeff <= 0 {
		t.Errorf("besselCoefficient should be positive, got %f", coeff)
	}

	// Test that coefficient exists for valid k within order
	coeff = besselCoefficient(3, 2)
	if coeff <= 0 {
		t.Errorf("besselCoefficient(3, 2) should be positive, got %f", coeff)
	}

	// All Bessel coefficients should be positive for valid parameters
	for order := 1; order <= 4; order++ {
		for k := 0; k <= order; k++ {
			coeff := besselCoefficient(order, k)
			if coeff <= 0 {
				t.Errorf("besselCoefficient(%d, %d) should be positive, got %f", order, k, coeff)
			}
		}
	}
}

// TestBesselAlignment tests Bessel alignment values
func TestBesselAlignment(t *testing.T) {
	// Test 3dB alignment (0.5)
	val := besselAlignment(2, 0.5)
	if math.Abs(val-1.36165412871613) > 1e-6 {
		t.Errorf("besselAlignment(2, 0.5) = %f, want 1.36165412871613", val)
	}

	// Test 6dB alignment (0.25)
	val = besselAlignment(1, 0.25)
	if math.Abs(val-1.73205080756888) > 1e-6 {
		t.Errorf("besselAlignment(1, 0.25) = %f, want 1.73205080756888", val)
	}

	// Test unsupported order/scale
	val = besselAlignment(20, 0.5)
	if val != 1.0 {
		t.Errorf("besselAlignment with unsupported values should return 1.0, got %f", val)
	}
}

// TestBesselPhaseMatched tests Bessel phase-matched values
func TestBesselPhaseMatched(t *testing.T) {
	tests := []struct {
		order int
		want  float64
	}{
		{1, 1.0000000232051},
		{2, 1.73205084237653},
		{3, 2.48134247792628},
		{8, 6.32439553519847},
		{10, 1.0}, // Unsupported order
	}

	for _, tt := range tests {
		got := besselPhaseMatched(tt.order)
		if math.Abs(got-tt.want) > 1e-6 {
			t.Errorf("besselPhaseMatched(%d) = %f, want %f", tt.order, got, tt.want)
		}
	}
}

// TestSallenKeyCoefficients tests Sallen-Key coefficient generation
func TestSallenKeyCoefficients(t *testing.T) {
	// Order 1
	coeffs := sallenKeyCoefficients(1, 0.707)
	if len(coeffs) != 2 {
		t.Errorf("sallenKeyCoefficients(1, 0.707) returned wrong length: got %d, want 2", len(coeffs))
	}

	// Order 2 with Q=0.707
	coeffs = sallenKeyCoefficients(2, 0.707)
	if len(coeffs) != 3 {
		t.Errorf("sallenKeyCoefficients(2, 0.707) returned wrong length: got %d, want 3", len(coeffs))
	}

	// First coefficient should be 1.0
	if coeffs[0] != 1.0 {
		t.Errorf("First coefficient should be 1.0, got %f", coeffs[0])
	}

	// Zero or negative Q should default to 1
	coeffs = sallenKeyCoefficients(2, -1)
	if coeffs[1] != 1.0 {
		t.Errorf("With negative Q, second coefficient should be 1.0, got %f", coeffs[1])
	}
}

// TestEvalAnalogResponse tests analog filter response evaluation
func TestEvalAnalogResponse(t *testing.T) {
	coeffs := []float64{1.0, 1.414, 1.0} // Butterworth order 2
	frequencies := []float64{100, 1000, 10000}
	freqCrit := 1000.0
	gain := 1.0

	// Lowpass response
	response := evalAnalogResponse(coeffs, frequencies, freqCrit, gain, false)
	if len(response) != len(frequencies) {
		t.Errorf("Response length mismatch: got %d, want %d", len(response), len(frequencies))
	}

	// At cutoff frequency, magnitude should be approximately -3dB
	mag := cmplxAbs(response[1])
	magDB := 20.0 * math.Log10(mag)
	if math.Abs(magDB-(-3.0)) > 0.5 {
		t.Errorf("At cutoff, magnitude = %f dB, want approximately -3 dB", magDB)
	}

	// Empty coefficients
	response = evalAnalogResponse(nil, frequencies, freqCrit, gain, false)
	if response != nil {
		t.Error("evalAnalogResponse with nil coeffs should return nil")
	}

	// Zero critical frequency
	response = evalAnalogResponse(coeffs, frequencies, 0, gain, false)
	if response != nil {
		t.Error("evalAnalogResponse with zero freqCrit should return nil")
	}
}

// TestCalculateIIRTransfer tests IIR filter transfer function calculation
func TestCalculateIIRTransfer(t *testing.T) {
	params := &gll.IIRFilterParams{
		FilterType:   gll.FilterTypeLowPass,
		FilterShape:  gll.FilterShapeButterworth,
		Order:        2,
		FreqCritInHz: 1000.0,
		Alignment:    gll.FilterAlignLevel3dB,
	}

	frequencies := []float64{100, 1000, 10000}

	response, ok := calculateIIRTransfer(params, frequencies)
	if !ok {
		t.Fatal("calculateIIRTransfer failed")
	}

	if len(response) != len(frequencies) {
		t.Errorf("Response length = %d, want %d", len(response), len(frequencies))
	}

	// At cutoff frequency (1kHz), magnitude should be approximately -3dB
	mag := cmplxAbs(response[1])
	magDB := 20.0 * math.Log10(mag)
	if math.Abs(magDB-(-3.0)) > 1.0 {
		t.Errorf("At cutoff, magnitude = %f dB, want approximately -3 dB", magDB)
	}

	// At low frequency (100Hz), magnitude should be close to 0dB (passband)
	mag = cmplxAbs(response[0])
	magDB = 20.0 * math.Log10(mag)
	if magDB < -1.0 {
		t.Errorf("In passband, magnitude = %f dB, should be close to 0 dB", magDB)
	}
}

// TestCalculateIIRTransferHighPass tests highpass filter
func TestCalculateIIRTransferHighPass(t *testing.T) {
	params := &gll.IIRFilterParams{
		FilterType:   gll.FilterTypeHighPass,
		FilterShape:  gll.FilterShapeButterworth,
		Order:        2,
		FreqCritInHz: 1000.0,
		Alignment:    gll.FilterAlignLevel3dB,
	}

	frequencies := []float64{100, 1000, 10000}

	response, ok := calculateIIRTransfer(params, frequencies)
	if !ok {
		t.Fatal("calculateIIRTransfer failed for highpass")
	}

	// At cutoff frequency, magnitude should be approximately -3dB
	mag := cmplxAbs(response[1])
	magDB := 20.0 * math.Log10(mag)
	if math.Abs(magDB-(-3.0)) > 1.0 {
		t.Errorf("At cutoff, magnitude = %f dB, want approximately -3 dB", magDB)
	}

	// At high frequency (10kHz), magnitude should be close to 0dB (passband)
	mag = cmplxAbs(response[2])
	magDB = 20.0 * math.Log10(mag)
	if magDB < -1.0 {
		t.Errorf("In passband, magnitude = %f dB, should be close to 0 dB", magDB)
	}
}

// TestCalculateIIRTransferBessel tests Bessel filter
func TestCalculateIIRTransferBessel(t *testing.T) {
	params := &gll.IIRFilterParams{
		FilterType:   gll.FilterTypeLowPass,
		FilterShape:  gll.FilterShapeBessel,
		Order:        2,
		FreqCritInHz: 1000.0,
		Alignment:    gll.FilterAlignPhaseMatched,
	}

	frequencies := []float64{100, 1000, 10000}

	response, ok := calculateIIRTransfer(params, frequencies)
	if !ok {
		t.Fatal("calculateIIRTransfer failed for Bessel")
	}

	if len(response) != len(frequencies) {
		t.Errorf("Response length = %d, want %d", len(response), len(frequencies))
	}
}

// TestCalculateIIRTransferLinkwitzRiley tests Linkwitz-Riley filter
func TestCalculateIIRTransferLinkwitzRiley(t *testing.T) {
	params := &gll.IIRFilterParams{
		FilterType:   gll.FilterTypeLowPass,
		FilterShape:  gll.FilterShapeLinkwitzRiley,
		Order:        4, // LR4
		FreqCritInHz: 1000.0,
		Alignment:    gll.FilterAlignLevel3dB,
	}

	frequencies := []float64{100, 1000, 10000}

	response, ok := calculateIIRTransfer(params, frequencies)
	if !ok {
		t.Fatal("calculateIIRTransfer failed for Linkwitz-Riley")
	}

	// At cutoff frequency, LR4 should be approximately -6dB
	mag := cmplxAbs(response[1])
	magDB := 20.0 * math.Log10(mag)
	if math.Abs(magDB-(-6.0)) > 1.0 {
		t.Errorf("LR4 at cutoff, magnitude = %f dB, want approximately -6 dB", magDB)
	}
}

// TestCalculateIIRTransferInvalidParams tests error handling
func TestCalculateIIRTransferInvalidParams(t *testing.T) {
	frequencies := []float64{100, 1000, 10000}

	// Nil params
	_, ok := calculateIIRTransfer(nil, frequencies)
	if ok {
		t.Error("calculateIIRTransfer with nil params should return false")
	}

	// Zero critical frequency
	params := &gll.IIRFilterParams{
		FilterType:   gll.FilterTypeLowPass,
		FilterShape:  gll.FilterShapeButterworth,
		Order:        2,
		FreqCritInHz: 0,
	}
	_, ok = calculateIIRTransfer(params, frequencies)
	if ok {
		t.Error("calculateIIRTransfer with zero freqCrit should return false")
	}

	// Empty frequencies
	params.FreqCritInHz = 1000
	_, ok = calculateIIRTransfer(params, nil)
	if ok {
		t.Error("calculateIIRTransfer with nil frequencies should return false")
	}

	// Unsupported filter shape
	params.FilterShape = 999
	_, ok = calculateIIRTransfer(params, frequencies)
	if ok {
		t.Error("calculateIIRTransfer with unsupported filter shape should return false")
	}

	// Unsupported filter type
	params.FilterShape = gll.FilterShapeButterworth
	params.FilterType = 999
	_, ok = calculateIIRTransfer(params, frequencies)
	if ok {
		t.Error("calculateIIRTransfer with unsupported filter type should return false")
	}
}

// TestBuildIIRResponse tests conversion to level/phase
func TestBuildIIRResponse(t *testing.T) {
	params := &gll.IIRFilterParams{
		FilterType:   gll.FilterTypeLowPass,
		FilterShape:  gll.FilterShapeButterworth,
		Order:        2,
		FreqCritInHz: 1000.0,
		Alignment:    gll.FilterAlignLevel3dB,
	}

	frequencies := []float64{100, 1000, 10000}

	levels, phase, ok := buildIIRResponse(params, frequencies)
	if !ok {
		t.Fatal("buildIIRResponse failed")
	}

	if len(levels) != len(frequencies) {
		t.Errorf("Levels length = %d, want %d", len(levels), len(frequencies))
	}

	if len(phase) != len(frequencies) {
		t.Errorf("Phase length = %d, want %d", len(phase), len(frequencies))
	}

	// At cutoff, level should be approximately -3dB
	if math.Abs(levels[1]-(-3.0)) > 1.0 {
		t.Errorf("At cutoff, level = %f dB, want approximately -3 dB", levels[1])
	}

	// Phase values should be in valid range
	for i, p := range phase {
		if p < -math.Pi || p > math.Pi {
			t.Errorf("Phase[%d] = %f, should be in range [-π, π]", i, p)
		}
	}
}

// TestBuildIIRResponseInvalid tests error cases
func TestBuildIIRResponseInvalid(t *testing.T) {
	frequencies := []float64{100, 1000, 10000}

	// Nil params
	_, _, ok := buildIIRResponse(nil, frequencies)
	if ok {
		t.Error("buildIIRResponse with nil params should return false")
	}

	// Empty frequencies
	params := &gll.IIRFilterParams{
		FilterType:   gll.FilterTypeLowPass,
		FilterShape:  gll.FilterShapeButterworth,
		Order:        2,
		FreqCritInHz: 1000.0,
	}
	_, _, ok = buildIIRResponse(params, nil)
	if ok {
		t.Error("buildIIRResponse with nil frequencies should return false")
	}

	// Zero critical frequency
	params.FreqCritInHz = 0
	_, _, ok = buildIIRResponse(params, frequencies)
	if ok {
		t.Error("buildIIRResponse with zero freqCrit should return false")
	}
}

// Helper function to calculate complex absolute value
func cmplxAbs(c complex128) float64 {
	r := real(c)
	i := imag(c)
	return math.Sqrt(r*r + i*i)
}
