package transport

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/schlunsen/codex-sdk-go/types"
)

// FindCLI searches for the codex executable in standard locations:
//
//  1. The CODEX_PATH environment variable, if set
//  2. PATH via exec.LookPath("codex")
//  3. Common global install locations (npm, homebrew, bun, cargo)
//
// Returns the resolved path or a *types.CLINotFoundError.
func FindCLI() (string, error) {
	if p := os.Getenv("CODEX_PATH"); p != "" {
		if isFile(p) {
			return p, nil
		}
	}

	if p, err := exec.LookPath(binaryName()); err == nil {
		return p, nil
	}

	for _, location := range candidateLocations() {
		p := expandHome(location)
		if isFile(p) {
			return p, nil
		}
	}

	return "", types.NewCLINotFoundError(
		"Codex CLI not found. Install with:\n" +
			"  npm install -g @openai/codex\n" +
			"  # or: brew install codex\n" +
			"\nIf already installed, add it to PATH or set CODEX_PATH,\n" +
			"or provide the path via CodexOptions:\n" +
			"  types.NewCodexOptions().WithCodexPath(\"/path/to/codex\")",
	)
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "codex.exe"
	}
	return "codex"
}

func candidateLocations() []string {
	name := binaryName()
	return []string{
		"~/.npm-global/bin/" + name,
		"/opt/homebrew/bin/" + name,
		"/usr/local/bin/" + name,
		"~/.local/bin/" + name,
		"~/.bun/bin/" + name,
		"~/.cargo/bin/" + name,
		"~/node_modules/.bin/" + name,
		"~/.yarn/bin/" + name,
	}
}

func isFile(path string) bool {
	info, err := os.Stat(path) //nolint:gosec // G703: statting candidate install paths is the purpose of this function
	return err == nil && !info.IsDir()
}

// expandHome expands a leading ~ to the user's home directory.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home := os.Getenv("HOME")
	if home == "" {
		if usr, err := user.Current(); err == nil {
			home = usr.HomeDir
		}
	}
	if home == "" {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}
