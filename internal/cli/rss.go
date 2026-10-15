package cli

import "math"

func normalizePeakRSS(value, scale int64) (int64, bool) {
	if value <= 0 || scale <= 0 || value > math.MaxInt64/scale {
		return 0, false
	}
	return value * scale, true
}
