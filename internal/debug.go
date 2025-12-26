package internal

import (
	"fmt"
	"os"
)

var debugEnabled bool

// SetDebug toggles debug logging for the internal package.
func SetDebug(enabled bool) {
	debugEnabled = enabled
}

func debugf(format string, args ...any) {
	if !debugEnabled {
		return
	}
	fmt.Fprintf(os.Stderr, "debug: "+format+"\n", args...)
}
