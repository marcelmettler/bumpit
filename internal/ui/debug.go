package ui

import (
	"fmt"
	"os"
	"time"
)

var debugLog *os.File

func initDebug() {
	if os.Getenv("CHOREKIT_DEBUG") == "" {
		return
	}
	f, err := os.Create("/tmp/chorekit-debug.log")
	if err == nil {
		debugLog = f
	}
}

func dbg(format string, args ...any) {
	if debugLog == nil {
		return
	}
	fmt.Fprintf(debugLog, "[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}
