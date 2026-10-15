package cli

import (
	"math"
	"testing"
)

func TestPeakRSSUnitConversionAndUnavailableValues(t *testing.T) {
	for _, tc := range []struct {
		value, scale, want int64
		ok                 bool
	}{
		{128, 1, 128, true}, {128, 1024, 131072, true}, {0, 1, 0, false}, {-1, 1024, 0, false}, {1, 0, 0, false}, {math.MaxInt64, 1024, 0, false},
	} {
		got, ok := normalizePeakRSS(tc.value, tc.scale)
		if got != tc.want || ok != tc.ok {
			t.Fatal(tc, got, ok)
		}
	}
	if c := newMetricCollector(false, nil); c != nil || c.coreObserver() != nil || c.finish() != nil {
		t.Fatal("disabled metrics allocated collector")
	}
}
