//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly && !windows

package output

import (
	"errors"
	"os"
)

// Unsupported lock hosts fail closed rather than inventing unsafe lock-file
// deletion or stale PID heuristics. Read-only checks do not need a writer lock.
func tryLock(*os.File) error { return errors.New("unsupported cooperative file locking") }
func unlock(*os.File)        {}
