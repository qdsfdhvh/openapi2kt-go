//go:build !darwin && !linux

package cli

func processPeakRSSBytes() (int64, bool) { return 0, false }
