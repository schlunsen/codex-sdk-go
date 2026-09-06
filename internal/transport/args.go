package transport

import (
	"fmt"

	"github.com/schlunsen/codex-sdk-go/internal/config"
	"github.com/schlunsen/codex-sdk-go/types"
)

// RunArgs describes a single `codex exec` invocation.
type RunArgs struct {
	// Input is the prompt written to stdin.
	Input string
	// ThreadID, when set, resumes an existing thread (`resume <id>`).
	ThreadID string
	// Images are local image paths passed via --image.
	Images []string
	// OutputSchemaFile is a path to a JSON schema passed via --output-schema.
	OutputSchemaFile string

	BaseURL string
	APIKey  string
	Thread  *types.ThreadOptions
}

// BuildArgs assembles the command-line arguments for `codex exec`. The
// argument order mirrors the official TypeScript SDK so that precedence
// between structured config, raw overrides, SDK-managed settings and
// thread-specific options is identical.
func BuildArgs(cfg types.ConfigObject, rawOverrides []string, args RunArgs) ([]string, error) {
	cmd := []string{"exec", "--experimental-json"}

	flattened, err := config.Flatten(cfg)
	if err != nil {
		return nil, err
	}
	for _, o := range flattened {
		cmd = append(cmd, "--config", o)
	}
	for _, o := range rawOverrides {
		cmd = append(cmd, "--config", o)
	}

	if args.BaseURL != "" {
		lit, err := config.ToTOMLValue(args.BaseURL, "openai_base_url")
		if err != nil {
			return nil, err
		}
		cmd = append(cmd, "--config", "openai_base_url="+lit)
	}

	t := args.Thread
	if t == nil {
		t = &types.ThreadOptions{}
	}

	if t.Model != "" {
		cmd = append(cmd, "--model", t.Model)
	}
	if t.ThreadSource != "" && args.ThreadID == "" {
		cmd = append(cmd, "--thread-source", t.ThreadSource)
	}
	if t.SandboxMode != "" {
		cmd = append(cmd, "--sandbox", string(t.SandboxMode))
	}
	if t.WorkingDirectory != "" {
		cmd = append(cmd, "--cd", t.WorkingDirectory)
	}
	for _, dir := range t.AdditionalDirectories {
		cmd = append(cmd, "--add-dir", dir)
	}
	if t.SkipGitRepoCheck {
		cmd = append(cmd, "--skip-git-repo-check")
	}
	if args.OutputSchemaFile != "" {
		cmd = append(cmd, "--output-schema", args.OutputSchemaFile)
	}
	if t.ModelReasoningEffort != "" {
		cmd = append(cmd, "--config", fmt.Sprintf("model_reasoning_effort=%q", string(t.ModelReasoningEffort)))
	}
	if t.NetworkAccessEnabled != nil {
		cmd = append(cmd, "--config", fmt.Sprintf("sandbox_workspace_write.network_access=%t", *t.NetworkAccessEnabled))
	}
	switch {
	case t.WebSearchMode != "":
		cmd = append(cmd, "--config", fmt.Sprintf("web_search=%q", string(t.WebSearchMode)))
	case t.WebSearchEnabled != nil && *t.WebSearchEnabled:
		cmd = append(cmd, "--config", `web_search="live"`)
	case t.WebSearchEnabled != nil && !*t.WebSearchEnabled:
		cmd = append(cmd, "--config", `web_search="disabled"`)
	}
	if t.ApprovalPolicy != "" {
		cmd = append(cmd, "--config", fmt.Sprintf("approval_policy=%q", string(t.ApprovalPolicy)))
	}

	if args.ThreadID != "" {
		cmd = append(cmd, "resume", args.ThreadID)
	}
	for _, img := range args.Images {
		cmd = append(cmd, "--image", img)
	}

	return cmd, nil
}
