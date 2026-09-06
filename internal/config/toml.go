// Package config serializes structured Codex config overrides into the
// `--config key=value` TOML-literal form the codex CLI expects.
package config

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/schlunsen/codex-sdk-go/types"
)

var tomlBareKey = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Flatten converts a nested config object into a list of `dotted.path=value`
// strings suitable for `--config`. Keys are emitted in sorted order for
// deterministic output.
func Flatten(cfg types.ConfigObject) ([]string, error) {
	if cfg == nil {
		return nil, nil
	}
	var out []string
	if err := flatten(cfg, "", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func flatten(value any, prefix string, out *[]string) error {
	m, ok := asMap(value)
	if !ok {
		if prefix == "" {
			return &types.ConfigError{Err: fmt.Errorf("config overrides must be a map")}
		}
		lit, err := ToTOMLValue(value, prefix)
		if err != nil {
			return err
		}
		*out = append(*out, prefix+"="+lit)
		return nil
	}

	keys := sortedKeys(m)
	if prefix == "" && len(keys) == 0 {
		return nil
	}
	if prefix != "" && len(keys) == 0 {
		*out = append(*out, prefix+"={}")
		return nil
	}

	for _, key := range keys {
		if key == "" {
			return &types.ConfigError{Path: prefix, Err: fmt.Errorf("config override keys must be non-empty strings")}
		}
		child := m[key]
		if child == nil {
			continue
		}
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		if _, isMap := asMap(child); isMap {
			if err := flatten(child, path, out); err != nil {
				return err
			}
			continue
		}
		lit, err := ToTOMLValue(child, path)
		if err != nil {
			return err
		}
		*out = append(*out, path+"="+lit)
	}
	return nil
}

// ToTOMLValue renders a Go value as a TOML literal. path is used for error
// messages only.
func ToTOMLValue(value any, path string) (string, error) {
	switch v := value.(type) {
	case nil:
		return "", &types.ConfigError{Path: path, Err: fmt.Errorf("value cannot be nil")}
	case string:
		return quoteString(v), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return formatFloat(float64(v), path)
	case float64:
		return formatFloat(v, path)
	case json.Number:
		return v.String(), nil
	}

	if m, ok := asMap(value); ok {
		keys := sortedKeys(m)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			if key == "" {
				return "", &types.ConfigError{Path: path, Err: fmt.Errorf("config override keys must be non-empty strings")}
			}
			child := m[key]
			if child == nil {
				continue
			}
			lit, err := ToTOMLValue(child, path+"."+key)
			if err != nil {
				return "", err
			}
			parts = append(parts, formatKey(key)+" = "+lit)
		}
		return "{" + strings.Join(parts, ", ") + "}", nil
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		parts := make([]string, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			lit, err := ToTOMLValue(rv.Index(i).Interface(), fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return "", err
			}
			parts = append(parts, lit)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	}

	return "", &types.ConfigError{Path: path, Err: fmt.Errorf("unsupported value type %T", value)}
}

func formatFloat(f float64, path string) (string, error) {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return "", &types.ConfigError{Path: path, Err: fmt.Errorf("must be a finite number")}
	}
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatFloat(f, 'f', 1, 64), nil
	}
	return strconv.FormatFloat(f, 'g', -1, 64), nil
}

// quoteString renders a TOML basic string. JSON string escaping is a valid
// subset of TOML basic-string escaping, which matches what the TypeScript SDK
// does via JSON.stringify.
func quoteString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func formatKey(key string) string {
	if tomlBareKey.MatchString(key) {
		return key
	}
	return quoteString(key)
}

func asMap(value any) (map[string]any, bool) {
	switch m := value.(type) {
	case map[string]any: // also covers types.ConfigObject (an alias)
		return m, true
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			out[iter.Key().String()] = iter.Value().Interface()
		}
		return out, true
	}
	return nil, false
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
