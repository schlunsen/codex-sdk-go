package codex

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version is the SDK version, read from the VERSION file at build time.
var Version = strings.TrimSpace(versionFile)
