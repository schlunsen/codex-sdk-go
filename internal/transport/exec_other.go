//go:build !unix

package transport

import (
	"context"
	"os/exec"
)

// configureProcessGroup is a no-op on platforms without POSIX process groups;
// cancellation falls back to os/exec's default Process.Kill.
func configureProcessGroup(_ *exec.Cmd) {}

// superviseProcessGroup is a no-op on platforms without POSIX process groups.
func superviseProcessGroup(_ context.Context, _ *exec.Cmd, _ <-chan struct{}) {}
