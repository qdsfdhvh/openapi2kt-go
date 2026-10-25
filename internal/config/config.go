// Package config reads TOML into the closed generator configuration schema.
// It performs no filesystem, environment or generator IO.
package config

import (
	"openapi-kt/internal/diagnostic"
	"strconv"
	"unicode/utf8"
)

const MaxBytes = 1024 * 1024
const MaxSpecs = 256

// Settings retains presence: an empty list/string or false replaces defaults.
type Settings map[string]any

// Spec is a named configuration entry; paths are not resolved by the decoder.
type Spec struct {
	Name     string
	Settings Settings
}
type Config struct {
	Defaults Settings
	Specs    []Spec
}

func problem(code string) *diagnostic.Diagnostic {
	return diagnostic.New(code, "Invalid generator configuration. See the documented configuration schema and TOML language.")
}
func owner(name string) bool {
	if len(name) == 0 || !(name[0] >= 'A' && name[0] <= 'Z' || name[0] >= 'a' && name[0] <= 'z') {
		return false
	}
	for _, c := range []byte(name) {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
func knownSetting(key string) bool {
	switch key {
	case "deny_warnings", "allow_warnings", "config_version", "name", "input", "input_format", "output_dir", "package_name", "model_name_suffix", "api_client_name", "runtime_package", "model_style", "exclude_apis", "exclude_merge_models", "exclude_merge_model_patterns", "exclude_merge_model_suffixes", "validation", "deduplicate_models", "merge_identical_models", "hoist_shared_shapes", "qualify_hoisted_names", "lock_hoisted_names":
		return true
	}
	return false
}
func typeProblem(expected string) *diagnostic.Diagnostic {
	d := problem("E_CONFIG_INVALID")
	d.Message = "Configuration field requires a TOML " + expected + "."
	return d
}
func typeProblemAt(expected, pointer string) *diagnostic.Diagnostic {
	d := typeProblem(expected)
	d.Pointer = &pointer
	return d
}
func setting(key string, value any, spec bool) *diagnostic.Diagnostic {
	switch key {
	case "name":
		if !spec {
			return problem("E_CONFIG_UNKNOWN_FIELD")
		}
		if s, ok := value.(string); !ok {
			return typeProblem("string")
		} else if !owner(s) {
			return problem("E_CONFIG_INVALID")
		}
	case "input":
		if s, ok := value.(string); !ok {
			return typeProblem("string")
		} else if s == "-" {
			return problem("E_CONFIG_INVALID")
		}
	case "output_dir", "package_name", "model_name_suffix", "api_client_name", "runtime_package":
		if _, ok := value.(string); !ok {
			return typeProblem("string")
		}
	case "model_style":
		s, ok := value.(string)
		if !ok {
			return typeProblem("string")
		}
		if s != "" && s != "data_class" && s != "plain_class" {
			return problem("E_CONFIG_INVALID")
		}
	case "input_format":
		s, ok := value.(string)
		if !ok {
			return typeProblem("string")
		}
		if s != "auto" && s != "json" && s != "yaml" {
			return problem("E_CONFIG_INVALID")
		}
	case "deny_warnings", "validation", "deduplicate_models", "merge_identical_models", "hoist_shared_shapes", "qualify_hoisted_names", "lock_hoisted_names":
		if _, ok := value.(bool); !ok {
			return typeProblem("boolean")
		}
	case "allow_warnings":
		codes, ok := value.([]string)
		if !ok {
			return typeProblem("string array")
		}
		for _, code := range codes {
			if !diagnostic.WarningCode(code) {
				return problem("E_CONFIG_INVALID")
			}
		}
	case "exclude_apis", "exclude_merge_models", "exclude_merge_model_patterns", "exclude_merge_model_suffixes":
		if _, ok := value.([]string); !ok {
			return typeProblem("string array")
		}
	default:
		return problem("E_CONFIG_UNKNOWN_FIELD")
	}
	return nil
}

// Parse accepts the library's TOML language within fixed resource limits, then
// validates the existing closed schema without coercions or implicit IO.
func Parse(data []byte) (Config, *diagnostic.Diagnostic) {
	out := Config{Defaults: Settings{}, Specs: []Spec{}}
	if len(data) > MaxBytes {
		return out, located("E_CONFIG_LIMIT", "")
	}
	if !utf8.Valid(data) {
		return out, located("E_CONFIG_SYNTAX", "")
	}
	root, source, err := decodeTOML(data)
	if err != nil {
		return out, err
	}
	for _, key := range source.sortedKeys(root, nil) {
		switch key {
		case "config_version":
			n, ok := root[key].(int64)
			if !ok {
				return out, typeProblemAt("integer", "/config_version")
			}
			if n != 1 {
				return out, located("E_CONFIG_VERSION", "/config_version")
			}
		case "defaults":
			table, ok := root[key].(map[string]any)
			if !ok {
				return out, typeProblemAt("table", "/defaults")
			}
			settings, err := readSettings(table, source, keyPath("defaults"), false)
			if err != nil {
				return out, err
			}
			out.Defaults = settings
		case "specs":
			entries, ok := root[key].([]any)
			if !ok {
				// A malformed table envelope can still declare unknown fields.
				// Preserve the unknown-field rejection for old specs.download
				// tables even when no [[specs]] inventory was declared.
				if table, ok := root[key].(map[string]any); ok {
					for _, field := range source.sortedKeys(table, keyPath("specs")) {
						if err := setting(field, table[field], true); err != nil && err.Code == "E_CONFIG_UNKNOWN_FIELD" {
							return out, located("E_CONFIG_UNKNOWN_FIELD", "/specs")
						}
					}
				}
				return out, typeProblemAt("array of tables", "/specs")
			}
			if len(entries) > MaxSpecs {
				return out, located("E_CONFIG_LIMIT", "/specs")
			}
			for i, entry := range entries {
				scope := arrayPath(keyPath("specs"), i)
				table, ok := entry.(map[string]any)
				if !ok {
					return out, typeProblemAt("table", safePointer(scope))
				}
				settings, err := readSettings(table, source, scope, true)
				if err != nil {
					return out, err
				}
				out.Specs = append(out.Specs, Spec{Settings: settings})
			}
		default:
			return out, located("E_CONFIG_UNKNOWN_FIELD", safePointer(keyPath(key)))
		}
	}
	if _, exists := root["config_version"]; !exists {
		return out, located("E_CONFIG_VERSION", "/config_version")
	}
	names := map[string]bool{}
	for i := range out.Specs {
		pointer := "/specs/" + strconv.Itoa(i) + "/name"
		v, ok := out.Specs[i].Settings["name"].(string)
		if !ok || !owner(v) {
			return out, located("E_CONFIG_INVALID", pointer)
		}
		if names[v] {
			return out, located("E_CONFIG_DUPLICATE", pointer)
		}
		names[v] = true
		out.Specs[i].Name = v
		delete(out.Specs[i].Settings, "name")
	}
	return out, nil
}

func readSettings(table map[string]any, source sourceIndex, scope configPath, spec bool) (Settings, *diagnostic.Diagnostic) {
	out := Settings{}
	for _, key := range source.sortedKeys(table, scope) {
		value := table[key]
		// Decode generic TOML arrays without coercing mixed/nested/scalar
		// values. Preserve the old []string representation, including [].
		if items, ok := value.([]any); ok {
			strings := make([]string, 0, len(items))
			for _, item := range items {
				text, ok := item.(string)
				if !ok {
					break
				}
				strings = append(strings, text)
			}
			if len(strings) == len(items) {
				value = strings
			}
		}
		if err := setting(key, value, spec); err != nil {
			p := safePointer(extendPath(scope, []string{key}))
			err.Pointer = &p
			return nil, err
		}
		out[key] = value
	}
	return out, nil
}

// Overlay returns a fresh presence-preserving defaults/spec combination.
func Overlay(defaults, spec Settings) Settings {
	out := Settings{}
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range spec {
		out[k] = v
	}
	return out
}
