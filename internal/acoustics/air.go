package acoustics

import "math"

// DefaultAirProperties returns standard air conditions (20°C, 50% humidity).
func DefaultAirProperties() (temperature, humidity, speed float64) {
	return 20.0, 0.5, 343.0
}

// AirLossPerMeter returns air absorption in dB/m at the given frequency.
// Based on simplified ISO 9613-1 model.
func AirLossPerMeter(freq, humidity float64) float64 {
	return 0.001 * math.Pow(freq/1000.0, 1.5) * (1.0 - humidity*0.5)
}
