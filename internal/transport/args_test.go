package transport

import (
	"reflect"
	"testing"

	"github.com/schlunsen/codex-sdk-go/types"
)

func boolPtr(b bool) *bool { return &b }

func TestBuildArgsMinimal(t *testing.T) {
	got, err := BuildArgs(nil, nil, RunArgs{Input: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"exec", "--experimental-json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestBuildArgsFull(t *testing.T) {
	th := types.NewThreadOptions().
		WithModel("gpt-5-codex").
		WithThreadSource("sdk").
		WithSandboxMode(types.SandboxWorkspaceWrite).
		WithWorkingDirectory("/repo").
		WithAdditionalDirectories("/a", "/b").
		WithSkipGitRepoCheck(true).
		WithModelReasoningEffort(types.ReasoningHigh).
		WithNetworkAccess(true).
		WithWebSearchMode(types.WebSearchLive).
		WithApprovalPolicy(types.ApprovalNever)

	got, err := BuildArgs(
		types.ConfigObject{"show_raw_agent_reasoning": true},
		[]string{`raw.key="v"`},
		RunArgs{
			BaseURL:          "https://example.test/v1",
			Thread:           th,
			OutputSchemaFile: "/tmp/schema.json",
			Images:           []string{"a.png"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"exec", "--experimental-json",
		"--config", "show_raw_agent_reasoning=true",
		"--config", `raw.key="v"`,
		"--config", `openai_base_url="https://example.test/v1"`,
		"--model", "gpt-5-codex",
		"--thread-source", "sdk",
		"--sandbox", "workspace-write",
		"--cd", "/repo",
		"--add-dir", "/a",
		"--add-dir", "/b",
		"--skip-git-repo-check",
		"--output-schema", "/tmp/schema.json",
		"--config", `model_reasoning_effort="high"`,
		"--config", "sandbox_workspace_write.network_access=true",
		"--config", `web_search="live"`,
		"--config", `approval_policy="never"`,
		"--image", "a.png",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestBuildArgsResumeSkipsThreadSource(t *testing.T) {
	th := types.NewThreadOptions().WithThreadSource("sdk")
	got, err := BuildArgs(nil, nil, RunArgs{Thread: th, ThreadID: "thread_abc"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"exec", "--experimental-json", "resume", "thread_abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestBuildArgsLegacyWebSearch(t *testing.T) {
	for _, tc := range []struct {
		enabled *bool
		want    string
	}{
		{boolPtr(true), `web_search="live"`},
		{boolPtr(false), `web_search="disabled"`},
	} {
		got, err := BuildArgs(nil, nil, RunArgs{Thread: &types.ThreadOptions{WebSearchEnabled: tc.enabled}})
		if err != nil {
			t.Fatal(err)
		}
		if got[len(got)-1] != tc.want {
			t.Errorf("got %v want trailing %s", got, tc.want)
		}
	}
	// Explicit mode wins over the legacy boolean.
	got, _ := BuildArgs(nil, nil, RunArgs{Thread: &types.ThreadOptions{WebSearchEnabled: boolPtr(true), WebSearchMode: types.WebSearchCached}})
	if got[len(got)-1] != `web_search="cached"` {
		t.Errorf("got %v", got)
	}
}

func TestBuildArgsBadConfig(t *testing.T) {
	if _, err := BuildArgs(types.ConfigObject{"x": struct{}{}}, nil, RunArgs{}); err == nil {
		t.Fatal("expected config error")
	}
}
