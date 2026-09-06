package types

import (
	"errors"
	"fmt"
	"strings"
)

// CLINotFoundError is returned when the codex executable cannot be located.
type CLINotFoundError struct {
	Message string
}

func (e *CLINotFoundError) Error() string { return e.Message }

// NewCLINotFoundError creates a CLINotFoundError with the given message.
func NewCLINotFoundError(message string) *CLINotFoundError {
	return &CLINotFoundError{Message: message}
}

// IsCLINotFoundError reports whether err is a CLINotFoundError.
func IsCLINotFoundError(err error) bool {
	var target *CLINotFoundError
	return errors.As(err, &target)
}

// ExecError is returned when the codex process exits unsuccessfully.
type ExecError struct {
	// ExitCode is the process exit code, or -1 if the process was killed by a signal.
	ExitCode int
	// Signal is the name of the signal that terminated the process, if any.
	Signal string
	// Stderr holds the captured stderr output of the process.
	Stderr string
	// Err is the underlying error from os/exec, if any.
	Err error
}

func (e *ExecError) Error() string {
	detail := fmt.Sprintf("code %d", e.ExitCode)
	if e.Signal != "" {
		detail = "signal " + e.Signal
	}
	msg := "codex exec exited with " + detail
	if s := strings.TrimSpace(e.Stderr); s != "" {
		msg += ": " + s
	}
	return msg
}

func (e *ExecError) Unwrap() error { return e.Err }

// IsExecError reports whether err is an ExecError.
func IsExecError(err error) bool {
	var target *ExecError
	return errors.As(err, &target)
}

// TurnFailedError is returned by Thread.Run when the agent reports a
// "turn.failed" event.
type TurnFailedError struct {
	Message string
}

func (e *TurnFailedError) Error() string { return "turn failed: " + e.Message }

// IsTurnFailedError reports whether err is a TurnFailedError.
func IsTurnFailedError(err error) bool {
	var target *TurnFailedError
	return errors.As(err, &target)
}

// ThreadStreamError is returned when the event stream emits a fatal "error"
// event.
type ThreadStreamError struct {
	Message string
}

func (e *ThreadStreamError) Error() string { return "codex stream error: " + e.Message }

// IsThreadStreamError reports whether err is a ThreadStreamError.
func IsThreadStreamError(err error) bool {
	var target *ThreadStreamError
	return errors.As(err, &target)
}

// ParseError is returned when a JSONL line from codex exec cannot be decoded.
type ParseError struct {
	Line string
	Err  error
}

func (e *ParseError) Error() string {
	line := e.Line
	if len(line) > 200 {
		line = line[:200] + "..."
	}
	return fmt.Sprintf("failed to parse codex event %q: %v", line, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

// IsParseError reports whether err is a ParseError.
func IsParseError(err error) bool {
	var target *ParseError
	return errors.As(err, &target)
}

// ConfigError is returned when config overrides cannot be serialized.
type ConfigError struct {
	Path string
	Err  error
}

func (e *ConfigError) Error() string {
	if e.Path == "" {
		return "invalid codex config override: " + e.Err.Error()
	}
	return fmt.Sprintf("invalid codex config override at %s: %v", e.Path, e.Err)
}

func (e *ConfigError) Unwrap() error { return e.Err }
