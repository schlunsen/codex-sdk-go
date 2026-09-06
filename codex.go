package codex

import (
	"github.com/schlunsen/codex-sdk-go/internal/transport"
	"github.com/schlunsen/codex-sdk-go/types"
)

// Codex is the entry point for interacting with the Codex agent. Create one
// with New, then use StartThread or ResumeThread.
type Codex struct {
	exec    *transport.Exec
	options *types.CodexOptions
}

// New creates a Codex client. Passing nil uses default options. The codex
// executable is located at construction time; a *types.CLINotFoundError is
// returned if it cannot be found.
func New(options *types.CodexOptions) (*Codex, error) {
	if options == nil {
		options = types.NewCodexOptions()
	}
	ex, err := transport.NewExec(options.CodexPathOverride, options)
	if err != nil {
		return nil, err
	}
	return &Codex{exec: ex, options: options}, nil
}

// ExecutablePath returns the resolved path of the codex binary.
func (c *Codex) ExecutablePath() string { return c.exec.ExecutablePath }

// StartThread starts a new conversation with the agent. Passing nil uses
// default thread options. The thread id is populated once the first turn
// starts.
func (c *Codex) StartThread(options *types.ThreadOptions) *Thread {
	if options == nil {
		options = types.NewThreadOptions()
	}
	return &Thread{exec: c.exec, options: c.options, threadOptions: options}
}

// ResumeThread resumes a previously started thread by id. Threads are
// persisted by codex in ~/.codex/sessions.
func (c *Codex) ResumeThread(id string, options *types.ThreadOptions) *Thread {
	t := c.StartThread(options)
	t.id = id
	return t
}
