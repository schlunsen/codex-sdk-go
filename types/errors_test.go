package types

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestErrorHelpers(t *testing.T) {
	wrapped := fmt.Errorf("ctx: %w", NewCLINotFoundError("nope"))
	if !IsCLINotFoundError(wrapped) {
		t.Error("IsCLINotFoundError")
	}
	if !IsExecError(&ExecError{ExitCode: 1}) || IsExecError(errors.New("x")) {
		t.Error("IsExecError")
	}
	if !IsTurnFailedError(&TurnFailedError{Message: "m"}) {
		t.Error("IsTurnFailedError")
	}
	if !IsThreadStreamError(&ThreadStreamError{Message: "m"}) {
		t.Error("IsThreadStreamError")
	}
	pe := &ParseError{Line: strings.Repeat("x", 300), Err: errors.New("bad")}
	if !IsParseError(pe) || !strings.Contains(pe.Error(), "...") {
		t.Error("ParseError")
	}
	ee := &ExecError{ExitCode: -1, Signal: "killed", Stderr: "  oops \n"}
	if got := ee.Error(); got != "codex exec exited with signal killed: oops" {
		t.Errorf("got %q", got)
	}
	ce := &ConfigError{Path: "a.b", Err: errors.New("bad")}
	if !strings.Contains(ce.Error(), "a.b") || !errors.Is(ce, ce.Err) {
		t.Error("ConfigError")
	}
}
