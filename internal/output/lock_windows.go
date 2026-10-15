//go:build windows

package output

import (
	"os"
	"syscall"
	"unsafe"
)

var kernel = syscall.NewLazyDLL("kernel32.dll")
var lockFile = kernel.NewProc("LockFileEx")
var unlockFile = kernel.NewProc("UnlockFileEx")

func tryLock(file *os.File) error {
	var overlap syscall.Overlapped
	// Nonblocking exclusive byte-zero lock overlaps the standard-library Rust
	// whole-file lock used by legacy cooperative writers.
	result, _, err := lockFile.Call(file.Fd(), 3, 0, 1, 0, uintptr(unsafe.Pointer(&overlap)))
	if result == 0 {
		return err
	}
	return nil
}
func unlock(file *os.File) {
	var overlap syscall.Overlapped
	_, _, _ = unlockFile.Call(file.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&overlap)))
}
