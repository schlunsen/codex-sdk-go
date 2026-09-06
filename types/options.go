package types

// ApprovalMode controls when Codex asks for approval before running commands.
type ApprovalMode string

const (
	ApprovalNever     ApprovalMode = "never"
	ApprovalOnRequest ApprovalMode = "on-request"
	ApprovalOnFailure ApprovalMode = "on-failure"
	ApprovalUntrusted ApprovalMode = "untrusted"
)

// SandboxMode controls the sandbox Codex runs commands in.
type SandboxMode string

const (
	SandboxReadOnly         SandboxMode = "read-only"
	SandboxWorkspaceWrite   SandboxMode = "workspace-write"
	SandboxDangerFullAccess SandboxMode = "danger-full-access"
)

// ModelReasoningEffort controls how much reasoning the model performs.
type ModelReasoningEffort string

const (
	ReasoningMinimal    ModelReasoningEffort = "minimal"
	ReasoningLow        ModelReasoningEffort = "low"
	ReasoningMedium     ModelReasoningEffort = "medium"
	ReasoningHigh       ModelReasoningEffort = "high"
	ReasoningXHigh      ModelReasoningEffort = "xhigh"
	ReasoningMax        ModelReasoningEffort = "max"
	ReasoningUltra      ModelReasoningEffort = "ultra"
	ReasoningPersistent ModelReasoningEffort = "persistent"
)

// WebSearchMode controls whether the agent may search the web.
type WebSearchMode string

const (
	WebSearchDisabled WebSearchMode = "disabled"
	WebSearchCached   WebSearchMode = "cached"
	WebSearchLive     WebSearchMode = "live"
)

// ConfigObject is a nested map of Codex configuration overrides. Values may be
// string, bool, any integer or float type, []any, or nested map[string]any.
// The SDK flattens it into dotted paths and serializes values as TOML
// literals, passing each as a `--config key=value` flag.
type ConfigObject = map[string]any

// CodexOptions configures the Codex client (the codex process itself).
type CodexOptions struct {
	// CodexPathOverride is an explicit path to the codex executable. When empty
	// the SDK searches PATH and common install locations.
	CodexPathOverride string
	// BaseURL, when set, is passed as `--config openai_base_url=...`.
	BaseURL string
	// APIKey, when set, is injected as CODEX_API_KEY into the process environment.
	APIKey string
	// Config holds structured `--config` overrides. See ConfigObject.
	Config ConfigObject
	// ConfigOverrides holds raw `--config key=value` strings forwarded unchanged
	// after Config and before SDK-managed / thread-specific overrides.
	ConfigOverrides []string
	// Env is the full environment for the codex process. When nil the SDK
	// inherits the current process environment. The SDK still injects its
	// required variables (such as CODEX_API_KEY) on top.
	Env map[string]string
	// Verbose enables debug logging to stderr.
	Verbose bool
}

// NewCodexOptions creates an empty CodexOptions for use with the builder methods.
func NewCodexOptions() *CodexOptions { return &CodexOptions{} }

// WithCodexPath sets an explicit path to the codex executable.
func (o *CodexOptions) WithCodexPath(path string) *CodexOptions {
	o.CodexPathOverride = path
	return o
}

// WithBaseURL sets the OpenAI-compatible base URL.
func (o *CodexOptions) WithBaseURL(url string) *CodexOptions {
	o.BaseURL = url
	return o
}

// WithAPIKey sets the API key passed to codex as CODEX_API_KEY.
func (o *CodexOptions) WithAPIKey(key string) *CodexOptions {
	o.APIKey = key
	return o
}

// WithConfig sets structured config overrides.
func (o *CodexOptions) WithConfig(cfg ConfigObject) *CodexOptions {
	o.Config = cfg
	return o
}

// WithConfigOverrides appends raw `--config key=value` overrides.
func (o *CodexOptions) WithConfigOverrides(overrides ...string) *CodexOptions {
	o.ConfigOverrides = append(o.ConfigOverrides, overrides...)
	return o
}

// WithEnv sets the full process environment (disables inheritance).
func (o *CodexOptions) WithEnv(env map[string]string) *CodexOptions {
	o.Env = env
	return o
}

// WithVerbose enables debug logging.
func (o *CodexOptions) WithVerbose(verbose bool) *CodexOptions {
	o.Verbose = verbose
	return o
}

// ThreadOptions configures a thread (conversation) with the agent.
type ThreadOptions struct {
	// Model overrides the model used for the thread (--model).
	Model string
	// ThreadSource is the source classification applied when the thread is
	// first created (--thread-source). Ignored when resuming.
	ThreadSource string
	// SandboxMode sets the sandbox policy (--sandbox).
	SandboxMode SandboxMode
	// WorkingDirectory sets the working directory for the agent (--cd).
	WorkingDirectory string
	// AdditionalDirectories grants the agent access to extra directories (--add-dir).
	AdditionalDirectories []string
	// SkipGitRepoCheck lets Codex run outside a git repository (--skip-git-repo-check).
	SkipGitRepoCheck bool
	// ModelReasoningEffort sets `--config model_reasoning_effort`.
	ModelReasoningEffort ModelReasoningEffort
	// NetworkAccessEnabled, when non-nil, sets `--config sandbox_workspace_write.network_access`.
	NetworkAccessEnabled *bool
	// WebSearchMode sets `--config web_search`.
	WebSearchMode WebSearchMode
	// WebSearchEnabled is the legacy boolean form of WebSearchMode. Ignored when
	// WebSearchMode is set.
	WebSearchEnabled *bool
	// ApprovalPolicy sets `--config approval_policy`.
	ApprovalPolicy ApprovalMode
}

// NewThreadOptions creates an empty ThreadOptions for use with the builder methods.
func NewThreadOptions() *ThreadOptions { return &ThreadOptions{} }

// WithModel sets the model.
func (o *ThreadOptions) WithModel(model string) *ThreadOptions {
	o.Model = model
	return o
}

// WithThreadSource sets the thread source classification.
func (o *ThreadOptions) WithThreadSource(source string) *ThreadOptions {
	o.ThreadSource = source
	return o
}

// WithSandboxMode sets the sandbox mode.
func (o *ThreadOptions) WithSandboxMode(mode SandboxMode) *ThreadOptions {
	o.SandboxMode = mode
	return o
}

// WithWorkingDirectory sets the working directory.
func (o *ThreadOptions) WithWorkingDirectory(dir string) *ThreadOptions {
	o.WorkingDirectory = dir
	return o
}

// WithAdditionalDirectories appends additional accessible directories.
func (o *ThreadOptions) WithAdditionalDirectories(dirs ...string) *ThreadOptions {
	o.AdditionalDirectories = append(o.AdditionalDirectories, dirs...)
	return o
}

// WithSkipGitRepoCheck toggles the git repository check.
func (o *ThreadOptions) WithSkipGitRepoCheck(skip bool) *ThreadOptions {
	o.SkipGitRepoCheck = skip
	return o
}

// WithModelReasoningEffort sets the reasoning effort.
func (o *ThreadOptions) WithModelReasoningEffort(effort ModelReasoningEffort) *ThreadOptions {
	o.ModelReasoningEffort = effort
	return o
}

// WithNetworkAccess enables or disables network access in workspace-write sandbox.
func (o *ThreadOptions) WithNetworkAccess(enabled bool) *ThreadOptions {
	o.NetworkAccessEnabled = &enabled
	return o
}

// WithWebSearchMode sets the web search mode.
func (o *ThreadOptions) WithWebSearchMode(mode WebSearchMode) *ThreadOptions {
	o.WebSearchMode = mode
	return o
}

// WithWebSearchEnabled sets the legacy web search boolean.
func (o *ThreadOptions) WithWebSearchEnabled(enabled bool) *ThreadOptions {
	o.WebSearchEnabled = &enabled
	return o
}

// WithApprovalPolicy sets the approval policy.
func (o *ThreadOptions) WithApprovalPolicy(policy ApprovalMode) *ThreadOptions {
	o.ApprovalPolicy = policy
	return o
}

// TurnOptions configures a single turn.
type TurnOptions struct {
	// OutputSchema is a JSON schema (as a plain map) describing the expected
	// agent output. When set, the SDK writes it to a temp file and passes
	// --output-schema.
	OutputSchema map[string]any
}

// NewTurnOptions creates an empty TurnOptions for use with the builder methods.
func NewTurnOptions() *TurnOptions { return &TurnOptions{} }

// WithOutputSchema sets the JSON schema for structured output.
func (o *TurnOptions) WithOutputSchema(schema map[string]any) *TurnOptions {
	o.OutputSchema = schema
	return o
}

// UserInputType discriminates UserInput entries.
type UserInputType string

const (
	UserInputText       UserInputType = "text"
	UserInputLocalImage UserInputType = "local_image"
)

// UserInput is a single input entry sent to the agent. Text entries are
// concatenated into the prompt; local_image entries are passed via --image.
type UserInput struct {
	Type UserInputType `json:"type"`
	Text string        `json:"text,omitempty"`
	Path string        `json:"path,omitempty"`
}

// TextInput creates a text input entry.
func TextInput(text string) UserInput {
	return UserInput{Type: UserInputText, Text: text}
}

// LocalImageInput creates a local image input entry.
func LocalImageInput(path string) UserInput {
	return UserInput{Type: UserInputLocalImage, Path: path}
}
