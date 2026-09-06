package config

import (
	"math"
	"reflect"
	"testing"

	"github.com/schlunsen/codex-sdk-go/types"
)

func TestFlattenNil(t *testing.T) {
	got, err := Flatten(nil)
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil; got %v, %v", got, err)
	}
	got, err = Flatten(types.ConfigObject{})
	if err != nil || len(got) != 0 {
		t.Fatalf("expected empty; got %v, %v", got, err)
	}
}

func TestFlattenNested(t *testing.T) {
	cfg := types.ConfigObject{
		"show_raw_agent_reasoning": true,
		"sandbox_workspace_write":  map[string]any{"network_access": true},
		"model":                    "gpt-5-codex",
		"retries":                  3,
		"temperature":              0.5,
		"tags":                     []any{"a", 1, true},
		"empty":                    map[string]any{},
		"skipped":                  nil,
		"weird key":                "x",
	}
	got, err := Flatten(cfg)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`empty={}`,
		`model="gpt-5-codex"`,
		`retries=3`,
		`sandbox_workspace_write.network_access=true`,
		`show_raw_agent_reasoning=true`,
		`tags=["a", 1, true]`,
		`temperature=0.5`,
		`weird key="x"`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}

func TestToTOMLValueInlineTable(t *testing.T) {
	got, err := ToTOMLValue([]any{map[string]any{"b": 2, "a key": "v"}}, "x")
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"a key" = "v", b = 2}]`
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestToTOMLValueErrors(t *testing.T) {
	cases := []any{nil, math.Inf(1), math.NaN(), struct{}{}, map[string]any{"": 1}}
	for _, c := range cases {
		if _, err := ToTOMLValue(c, "p"); err == nil {
			t.Errorf("expected error for %#v", c)
		}
	}
}

func TestStringEscaping(t *testing.T) {
	got, err := ToTOMLValue("a\"b\\c\n", "p")
	if err != nil {
		t.Fatal(err)
	}
	if got != `"a\"b\\c\n"` {
		t.Fatalf("got %s", got)
	}
}

func TestTypedMap(t *testing.T) {
	got, err := Flatten(types.ConfigObject{"m": map[string]string{"k": "v"}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{`m.k="v"`}) {
		t.Fatalf("got %#v", got)
	}
}
