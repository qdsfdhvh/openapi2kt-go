//go:build darwin || linux

package cli

import (
	"runtime"
	"syscall"
)

// Getrusage's high-water mark covers this process lifetime, not this invocation
// or a core phase. Darwin reports bytes; Linux reports KiB. No child/subprocess
// memory is included. Other hosts omit the field instead of approximating it.
func processPeakRSSBytes() (int64, bool) {
	var usage syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &usage) != nil {
		return 0, false
	}
	scale := int64(1)
	if runtime.GOOS == "linux" {
		scale = 1024
	}
	return normalizePeakRSS(int64(usage.Maxrss), scale)
}
