package filters

import (
	"math"
	"math/cmplx"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

func buildIIRResponse(params *gll.IIRFilterParams, frequencies []float64) ([]float64, []float64, bool) {
	if params == nil || len(frequencies) == 0 || params.FreqCritInHz <= 0 {
		return nil, nil, false
	}

	response, ok := calculateIIRTransfer(params, frequencies)
	if !ok || len(response) == 0 {
		return nil, nil, false
	}

	levels := make([]float64, len(response))
	phase := make([]float64, len(response))
	for i, value := range response {
		levels[i] = magnitudeToDB(cmplx.Abs(value))
		phase[i] = cmplx.Phase(value)
	}
	return levels, phase, true
}

func calculateIIRTransfer(params *gll.IIRFilterParams, frequencies []float64) ([]complex128, bool) {
	if params == nil || params.FreqCritInHz <= 0 || len(frequencies) == 0 {
		return nil, false
	}

	order := int(params.Order)
	if order <= 0 {
		order = 1
	}

	alignScale := 1.0
	phaseMatched := false
	switch params.Alignment {
	case gll.FilterAlignLevel3dB:
		alignScale = 0.5
	case gll.FilterAlignLevel6dB:
		alignScale = 0.25
	case gll.FilterAlignPhaseMatched:
		phaseMatched = true
	}

	num := 1.0
	gainScale := 1.0
	linkwitzRiley := false
	var coeffs []float64

	switch params.FilterShape {
	case gll.FilterShapeButterworth:
		num = butterworthNormalization(order, alignScale)
		coeffs = butterworthCoefficients(order)
	case gll.FilterShapeBessel:
		if alignScale != 1.0 {
			num = besselAlignment(order, alignScale)
		} else if phaseMatched {
			num = besselPhaseMatched(order)
		}
		coeffs = besselCoefficients(order)
	case gll.FilterShapeLinkwitzRiley:
		linkwitzRiley = true
		gainScale = 0.5
		halfOrder := order / 2
		if halfOrder < 1 {
			halfOrder = 1
		}
		num = butterworthNormalization(halfOrder, 0.5)
		coeffs = butterworthCoefficients(halfOrder)
	case gll.FilterShapeSallenKey:
		coeffs = sallenKeyCoefficients(order, params.QFactor)
	default:
		return nil, false
	}

	if len(coeffs) == 0 {
		return nil, false
	}

	var response []complex128
	switch params.FilterType {
	case gll.FilterTypeLowPass:
		response = evalAnalogResponse(coeffs, frequencies, params.FreqCritInHz, num, false)
	case gll.FilterTypeHighPass:
		response = evalAnalogResponse(coeffs, frequencies, params.FreqCritInHz, num, true)
	case gll.FilterTypeAllPass:
		a := evalAnalogResponse(coeffs, frequencies, params.FreqCritInHz, num, false)
		b := evalAnalogResponse(coeffs, frequencies, params.FreqCritInHz, -num, false)
		response = complexDivide(a, b)
	case gll.FilterTypePeak, gll.FilterTypePeakSym:
		a := evalAnalogResponse(coeffs, frequencies, params.FreqCritInHz, num, false)
		b := evalAnalogResponse(coeffs, frequencies, params.FreqCritInHz, num, true)
		response = complexAdd(a, b)
		response = complexScale(response, -1.0)
		response = complexAddScalar(response, 1.0)
		if params.FilterType == gll.FilterTypePeakSym && params.ParametricGainIndB < 0 {
			gain := math.Pow(10.0, gainScale*(-params.ParametricGainIndB)/20.0)
			response = complexScale(response, gain-1.0)
			response = complexAddScalar(response, 1.0)
		} else {
			gain := math.Pow(10.0, gainScale*params.ParametricGainIndB/20.0)
			response = complexScale(response, gain-1.0)
			response = complexAddScalar(response, 1.0)
		}
	case gll.FilterTypeLowShelf, gll.FilterTypeHighShelf:
		a := evalAnalogResponse(coeffs, frequencies, params.FreqCritInHz, num, false)
		b := evalAnalogResponse(coeffs, frequencies, params.FreqCritInHz, num, true)
		gain := math.Pow(10.0, gainScale*params.ParametricGainIndB/20.0)
		if params.FilterType == gll.FilterTypeLowShelf {
			a = complexScale(a, gain)
			if order == 2 {
				b = complexScale(b, -1.0)
			}
		} else {
			if order == 2 {
				a = complexScale(a, -1.0)
			}
			b = complexScale(b, gain)
		}
		response = complexAdd(a, b)
	default:
		return nil, false
	}

	if linkwitzRiley {
		for i := range response {
			response[i] *= response[i]
		}
	}

	return response, true
}

func evalAnalogResponse(coeffs []float64, frequencies []float64, freqCrit, gain float64, invert bool) []complex128 {
	if len(coeffs) == 0 || freqCrit <= 0 {
		return nil
	}
	const minRatio = 1e-9
	response := make([]complex128, len(frequencies))
	base := complex(coeffs[0], 0)
	for i, freq := range frequencies {
		ratio := freq / freqCrit
		if ratio < minRatio {
			ratio = minRatio
		}
		num := gain
		if invert {
			num = (-num) / ratio
		} else {
			num *= ratio
		}
		s := complex(0, num)
		denom := base
		power := complex(1, 0)
		for j := 1; j < len(coeffs); j++ {
			power *= s
			denom += complex(coeffs[j], 0) * power
		}
		response[i] = base / denom
	}
	return response
}

func complexAdd(a, b []complex128) []complex128 {
	if len(a) == 0 || len(a) != len(b) {
		return nil
	}
	out := make([]complex128, len(a))
	for i := range a {
		out[i] = a[i] + b[i]
	}
	return out
}

func complexDivide(a, b []complex128) []complex128 {
	if len(a) == 0 || len(a) != len(b) {
		return nil
	}
	out := make([]complex128, len(a))
	for i := range a {
		out[i] = a[i] / b[i]
	}
	return out
}

func complexScale(a []complex128, scale float64) []complex128 {
	if len(a) == 0 {
		return nil
	}
	out := make([]complex128, len(a))
	factor := complex(scale, 0)
	for i := range a {
		out[i] = a[i] * factor
	}
	return out
}

func complexAddScalar(a []complex128, scalar float64) []complex128 {
	if len(a) == 0 {
		return nil
	}
	out := make([]complex128, len(a))
	add := complex(scalar, 0)
	for i := range a {
		out[i] = a[i] + add
	}
	return out
}

func butterworthNormalization(order int, alignScale float64) float64 {
	if order <= 0 || alignScale <= 0 {
		return 1.0
	}
	num := math.Sqrt(1.0/alignScale - 1.0)
	num2 := math.Pow(1.0/float64(order), 1.0/float64(order))
	num3 := math.Pow(1.0/num, 1.0/float64(order))
	return num2 / num3
}

func butterworthCoefficients(order int) []float64 {
	if order <= 0 {
		return nil
	}
	coeffs := make([]float64, order+1)
	for i := 0; i <= order; i++ {
		coeffs[i] = butterworthCoefficient(order, order-i)
	}
	return coeffs
}

func butterworthCoefficient(order, k int) float64 {
	if order <= 0 || k < 0 || k > order {
		return 0
	}
	num := math.Pow(1.0/float64(order), 1.0/float64(order))
	values := make([]float64, k+1)
	values[0] = 1.0
	for i := 1; i <= k; i++ {
		angle := float64(i-1) * math.Pi / (2.0 * float64(order))
		values[i] = values[i-1] * num * math.Cos(angle) / math.Sin(float64(i)*math.Pi/(2.0*float64(order)))
	}
	return values[k]
}

func besselCoefficients(order int) []float64 {
	if order <= 0 {
		return nil
	}
	coeffs := make([]float64, order+1)
	for i := 0; i <= order; i++ {
		coeffs[i] = besselCoefficient(order, i)
	}
	return coeffs
}

func besselCoefficient(order, k int) float64 {
	return factorial(2*order-k) / (math.Pow(2, float64(order-k)) * factorial(k) * factorial(order-k))
}

func besselAlignment(order int, alignScale float64) float64 {
	switch alignScale {
	case 0.5:
		switch order {
		case 1:
			return 1.0
		case 2:
			return 1.36165412871613
		case 3:
			return 1.75567236868121
		case 4:
			return 2.11391767490422
		case 5:
			return 2.42741070215263
		case 6:
			return 2.70339506120292
		case 7:
			return 2.95172214703872
		case 8:
			return 3.17961723751065
		}
	case 0.25:
		switch order {
		case 1:
			return 1.73205080756888
		case 2:
			return 1.97694888987955
		case 3:
			return 2.42454770439973
		case 4:
			return 2.88602284792378
		case 5:
			return 3.32415542718002
		case 6:
			return 3.72655755891719
		case 7:
			return 4.09207415068004
		case 8:
			return 4.42556630305568
		}
	}
	return 1.0
}

func besselPhaseMatched(order int) float64 {
	switch order {
	case 1:
		return 1.0000000232051
	case 2:
		return 1.73205084237653
	case 3:
		return 2.48134247792628
	case 4:
		return 3.24037034920393
	case 5:
		return 4.00574980621619
	case 6:
		return 4.77560085578494
	case 7:
		return 5.5487473277673
	case 8:
		return 6.32439553519847
	default:
		return 1.0
	}
}

func sallenKeyCoefficients(order int, qFactor float64) []float64 {
	if order <= 1 {
		return []float64{1.0, 1.0}
	}
	if qFactor <= 0 {
		qFactor = 1
	}
	return []float64{1.0, 1.0 / qFactor, 1.0}
}

func factorial(value int) float64 {
	if value <= 1 {
		return 1
	}
	out := 1.0
	for i := 2; i <= value; i++ {
		out *= float64(i)
	}
	return out
}
