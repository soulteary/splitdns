package system

import (
	"os"
	"runtime"
)

func goos() string {
	return runtime.GOOS
}

// isTerminal reports whether w is a character device (a TTY).
func isTerminal(w *os.File) bool {
	info, err := w.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
