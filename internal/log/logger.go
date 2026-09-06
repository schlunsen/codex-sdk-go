// Package log provides a minimal verbosity-gated logger for the SDK.
package log

import (
	stdlog "log"
	"os"
)

// Logger writes debug output to stderr when verbose is enabled.
type Logger struct {
	verbose bool
	l       *stdlog.Logger
}

// NewLogger creates a Logger. When verbose is false all Debugf calls are no-ops.
func NewLogger(verbose bool) *Logger {
	return &Logger{verbose: verbose, l: stdlog.New(os.Stderr, "[codex-sdk] ", stdlog.LstdFlags|stdlog.Lmicroseconds)}
}

// Debugf logs a formatted message when verbose is enabled.
func (l *Logger) Debugf(format string, args ...any) {
	if l == nil || !l.verbose {
		return
	}
	l.l.Printf(format, args...)
}
