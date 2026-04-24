package acoustics

import "math"

const (
	ReferenceTemperatureK = 293.15
	TriplePointK          = 273.16
	ReferencePressureKPa  = 101.325
)

// DefaultAirProperties returns standard air conditions (20°C, 50% humidity).
func DefaultAirProperties() (temperature, humidity, speed float64) {
	return 20.0, 0.5, 343.0
}

// AirLossPerMeter returns air absorption in dB/m at the given frequency.
// Based on ISO 9613-1 atmospheric absorption for pure-tone sound.
func AirLossPerMeter(freq, temperatureC, humidity, pressureKPa float64) float64 {
	if freq <= 0 {
		return 0
	}
	if pressureKPa <= 0 {
		pressureKPa = ReferencePressureKPa
	}
	if humidity < 0 {
		humidity = 0
	} else if humidity > 1 {
		humidity = 1
	}

	temperatureK := temperatureC + 273.15
	if temperatureK <= 0 {
		temperatureK = ReferenceTemperatureK
	}

	pressureRatio := pressureKPa / ReferencePressureKPa
	temperatureRatio := temperatureK / ReferenceTemperatureK
	molarHumidity := humidity * saturationVaporPressureRatio(temperatureK) / pressureRatio

	oxygenRelaxation := pressureRatio * (24.0 + 4.04e4*molarHumidity*(0.02+molarHumidity)/(0.391+molarHumidity))
	nitrogenRelaxation := pressureRatio * math.Pow(temperatureRatio, -0.5) *
		(9.0 + 280.0*molarHumidity*math.Exp(-4.17*(math.Pow(temperatureRatio, -1.0/3.0)-1.0)))

	classical := 1.84e-11 * math.Pow(pressureRatio, -1) * math.Sqrt(temperatureRatio)
	oxygen := 0.01275 * math.Exp(-2239.1/temperatureK) / (oxygenRelaxation + freq*freq/oxygenRelaxation)
	nitrogen := 0.1068 * math.Exp(-3352.0/temperatureK) / (nitrogenRelaxation + freq*freq/nitrogenRelaxation)
	molecular := math.Pow(temperatureRatio, -2.5) * (oxygen + nitrogen)

	attenuationDBPerMeter := 8.686 * freq * freq * (classical + molecular)
	if math.IsNaN(attenuationDBPerMeter) || math.IsInf(attenuationDBPerMeter, 0) || attenuationDBPerMeter < 0 {
		return 0
	}
	return attenuationDBPerMeter
}

func saturationVaporPressureRatio(temperatureK float64) float64 {
	return math.Pow(10, -6.8346*math.Pow(TriplePointK/temperatureK, 1.261)+4.6151)
}
