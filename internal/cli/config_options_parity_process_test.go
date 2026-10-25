package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"openapi-kt/core"
)

func paritySettings(o core.GenerationOptions) map[string]any {
	return map[string]any{
		"package_name": o.PackageName, "runtime_package": o.RuntimePackage,
		"model_name_suffix": o.ModelNameSuffix, "model_style": string(o.ModelStyle),
		"api_client_name": o.APIClientName, "exclude_apis": o.ExcludeAPIs,
		"deduplicate_models": o.DeduplicateModels, "merge_identical_models": o.MergeIdenticalModels,
		"exclude_merge_models": o.ExcludeMergeModels, "exclude_merge_model_patterns": o.ExcludeMergeModelPatterns,
		"exclude_merge_model_suffixes": o.ExcludeMergeModelSuffixes,
		"hoist_shared_shapes":          o.HoistSharedShapes, "qualify_hoisted_names": o.QualifyHoistedNames,
		"lock_hoisted_names": o.LockHoistedNames, "validation": o.Validation,
	}
}

func parityTable(settings map[string]any) string {
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, key := range keys {
		value := settings[key]
		if items, ok := value.([]string); ok && items == nil {
			value = []string{}
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			panic(err) // Fixed synthetic string/bool/string-array settings only.
		}
		fmt.Fprintf(&out, "%s=%s\n", key, encoded)
	}
	return out.String()
}

func parityDirectArguments(input, output string, format core.InputFormat, o core.GenerationOptions) []string {
	args := []string{"generate", "--input", input, "--output", output, "--input-format=" + string(format), "--diagnostic-format=json"}
	flags := map[string]string{
		"package_name": "--package-name", "runtime_package": "--runtime-package",
		"model_name_suffix": "--model-name-suffix", "model_style": "--model-style",
		"api_client_name": "--api-client-name", "exclude_apis": "--exclude-api",
		"deduplicate_models": "--deduplicate-models", "merge_identical_models": "--merge-identical-models",
		"exclude_merge_models": "--exclude-merge-model", "exclude_merge_model_patterns": "--exclude-merge-model-pattern",
		"exclude_merge_model_suffixes": "--exclude-merge-model-suffix", "hoist_shared_shapes": "--hoist-shared-shapes",
		"qualify_hoisted_names": "--qualify-hoisted-names", "lock_hoisted_names": "--lock-hoisted-names", "validation": "--validation",
	}
	settings := paritySettings(o)
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		flag := flags[key]
		switch value := settings[key].(type) {
		case string:
			if value != "" {
				args = append(args, flag+"="+value)
			}
		case bool:
			if key == "validation" {
				if value {
					args = append(args, flag)
				}
			} else {
				args = append(args, fmt.Sprintf("%s=%t", flag, value))
			}
		case []string:
			for _, item := range value {
				args = append(args, flag+"="+item)
			}
		}
	}
	return args
}

func TestConfigOptionsPublicFieldInventory(t *testing.T) {
	// A new public generation option needs an explicit configuration contract
	// and a behavior case below instead of silently inheriting a Go zero value.
	covered := map[string]bool{
		"PackageName": true, "RuntimePackage": true, "ModelNameSuffix": true, "ModelStyle": true,
		"DeduplicateModels": true, "MergeIdenticalModels": true, "ExcludeMergeModels": true,
		"ExcludeMergeModelPatterns": true, "ExcludeMergeModelSuffixes": true, "HoistSharedShapes": true,
		"QualifyHoistedNames": true, "LockHoistedNames": true, "APIClientName": true,
		"ExcludeAPIs": true, "Validation": true,
	}
	options := reflect.TypeOf(core.GenerationOptions{})
	for i := 0; i < options.NumField(); i++ {
		if !covered[options.Field(i).Name] {
			t.Fatal("public option has no configuration parity case", options.Field(i).Name)
		}
		delete(covered, options.Field(i).Name)
	}
	if len(covered) != 0 {
		t.Fatal("stale option coverage inventory", covered)
	}
}

func TestConfigOptionsNativeCoreAndDirectParity(t *testing.T) {
	baseOptions := core.GenerationOptions{PackageName: "org.example.parity", ModelNameSuffix: "Dto", APIClientName: "ConfiguredClient"}
	all := baseOptions
	all.RuntimePackage = "org.example.parity.shared"
	all.DeduplicateModels, all.MergeIdenticalModels, all.HoistSharedShapes, all.QualifyHoistedNames, all.LockHoistedNames, all.Validation = true, true, true, true, true, true
	all.ExcludeAPIs = []string{"DefaultApi"}
	all.ExcludeMergeModels = []string{"org.example.parity.models.ZetaDto"}
	all.ExcludeMergeModelPatterns = []string{".*ChildZDto"}
	all.ExcludeMergeModelSuffixes = []string{"ParentZDto"}
	dedup := baseOptions
	dedup.DeduplicateModels, dedup.Validation = true, true
	hoist := baseOptions
	hoist.HoistSharedShapes, hoist.QualifyHoistedNames, hoist.LockHoistedNames = true, true, true
	hoist.Validation = true
	runtime := baseOptions
	runtime.RuntimePackage = "org.example.parity.shared"
	merge := baseOptions
	merge.MergeIdenticalModels = true
	merge.ExcludeMergeModels = []string{"org.example.parity.models.ExactZDto"}
	merge.ExcludeMergeModelPatterns = []string{`org\.example\.parity\.models\.PatternZDto`}
	merge.ExcludeMergeModelSuffixes = []string{"SuffixZDto"}
	runtimeDoc := []byte(`{"openapi":"3.1.0","info":{"title":"Parity","version":"synthetic"},"paths":{"/update":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"value":{"type":"string","nullable":true},"raw":{"type":"object"}}}}}},"responses":{"204":{}}}}}}`)
	mergeDoc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"ControlA":{"properties":{"control":{"type":"string"}}},"ControlZ":{"properties":{"control":{"type":"string"}}},"ExactA":{"properties":{"exact":{"type":"string"}}},"ExactZ":{"properties":{"exact":{"type":"string"}}},"PatternA":{"properties":{"pattern":{"type":"string"}}},"PatternZ":{"properties":{"pattern":{"type":"string"}}},"SuffixA":{"properties":{"suffix":{"type":"string"}}},"SuffixZ":{"properties":{"suffix":{"type":"string"}}}}}}`)
	cases := []struct {
		name, fixture string
		document      []byte
		format        core.InputFormat
		options       core.GenerationOptions
	}{
		{"all-fields", "merge-models.json", nil, core.JSON, all},
		{"dedup-only", "deduplicate-models.json", nil, core.JSON, dedup},
		{"qualified-hoist-lock", "deduplicate-models.json", nil, core.JSON, hoist},
		{"runtime-patch", "", runtimeDoc, core.JSON, runtime},
		{"merge-exclusions", "", mergeDoc, core.JSON, merge},
		{"yaml-extensionless", "", []byte("openapi: 3.1.0\ncomponents:\n  schemas:\n    Value:\n      properties:\n        name:\n          type: string\n"), core.YAML, baseOptions},
	}
	for _, tc := range cases {
		for _, inDefaults := range []bool{true, false} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				t.Run(fmt.Sprintf("%s/defaults=%t/%s", tc.name, inDefaults, style), func(t *testing.T) {
					doc := tc.document
					if tc.fixture != "" {
						doc = read(t, filepath.Join(repo, "tests/fixtures/http", tc.fixture))
					}
					o := tc.options
					o.ModelStyle = style
					base := t.TempDir()
					input, cfg := filepath.Join(base, "source.data"), filepath.Join(base, "config.toml")
					put(t, input, doc)
					defaults := map[string]any{"input": "missing.data", "output_dir": "missing-output"}
					spec := map[string]any{"name": "catalog", "input": "source.data", "output_dir": "configured", "input_format": string(tc.format)}
					for key, value := range paritySettings(o) {
						if inDefaults {
							defaults[key] = value
						} else {
							spec[key] = value
						}
					}
					put(t, cfg, []byte("config_version=1\n[defaults]\n"+parityTable(defaults)+"[[specs]]\n"+parityTable(spec)))
					verifyConfigOptionParity(t, base, input, cfg, doc, tc.format, o)
				})
			}
		}
	}
}

func verifyConfigOptionParity(t *testing.T, base, input, cfg string, document []byte, format core.InputFormat, o core.GenerationOptions) {
	t.Helper()
	want, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Format: format, Options: o})
	if err != nil {
		t.Fatal("invalid synthetic parity control", err)
	}
	directRoot, configRoot := filepath.Join(base, "direct"), filepath.Join(base, "configured")
	directArgs := parityDirectArguments(input, directRoot, format, o)
	configArgs := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	for _, command := range [][]string{directArgs, configArgs} {
		got := result(t, invoke(t, t.TempDir(), command...), 0)
		if !reflect.DeepEqual(append(core.Diagnostics{}, got.Warnings...), append(core.Diagnostics{}, want.Warnings...)) || len(got.Files) != len(want.Artifacts.Files) {
			t.Fatal("adapter changed complete warnings or artifact inventory", got)
		}
	}
	for _, root := range []string{directRoot, configRoot} {
		var manifest struct{ Version int }
		if err := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &manifest); err != nil {
			t.Fatal(err)
		}
		version := 1
		if root == configRoot {
			version = 2
		}
		if o.LockHoistedNames {
			version += 2
		}
		if manifest.Version != version {
			t.Fatal("configuration adapter changed the lock/ownership mode", manifest.Version, version)
		}
		for _, file := range want.Artifacts.Files {
			if string(read(t, filepath.Join(root, file.Path))) != file.Content {
				t.Fatal("config/direct/core Kotlin differs", file.Path)
			}
		}
		if o.LockHoistedNames {
			path := ".openapi-kt-names.json"
			if root == configRoot {
				path = namedLockPath("catalog")
			}
			if !bytes.Equal(read(t, filepath.Join(root, path)), want.Artifacts.NameLock) {
				t.Fatal("config/direct/core name-lock semantics differ")
			}
		}
	}
	before := snapshot(t, configRoot)
	result(t, invoke(t, t.TempDir(), append(append([]string{}, configArgs...), "--check")...), 0)
	result(t, invoke(t, t.TempDir(), "validate", "--config", cfg, "--diagnostic-format=json"), 0)
	result(t, invoke(t, t.TempDir(), configArgs...), 0)
	if !reflect.DeepEqual(before, snapshot(t, configRoot)) {
		t.Fatal("config parity readonly/noop changed bytes/attributes")
	}
}

func TestConfigOptionsFalseEmptyOverridesNative(t *testing.T) {
	base := t.TempDir()
	input, cfg := filepath.Join(base, "source.json"), filepath.Join(base, "config.toml")
	document := []byte(`{"openapi":"3.1.0","info":{"title":"Parity","version":"synthetic"},"paths":{"/update":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"value":{"type":"string","nullable":true},"raw":{"type":"object"}}}}}},"responses":{"204":{}}}}}}`)
	put(t, input, document)
	defaults := paritySettings(core.GenerationOptions{PackageName: "org.example.inherited", RuntimePackage: "org.example.shared", ModelNameSuffix: "Dto", APIClientName: "InheritedClient", ModelStyle: core.PlainClass, Validation: true, DeduplicateModels: true, MergeIdenticalModels: true, HoistSharedShapes: true, QualifyHoistedNames: true, LockHoistedNames: true, ExcludeAPIs: []string{"DefaultApi"}, ExcludeMergeModels: []string{"org.example.inherited.models.KeepDto"}, ExcludeMergeModelPatterns: []string{".*KeepDto"}, ExcludeMergeModelSuffixes: []string{"KeepDto"}})
	defaults["deny_warnings"], defaults["allow_warnings"] = true, []string{"W_JSON_ELEMENT_APPROXIMATION"}
	defaults["input_format"] = "yaml"
	want := core.GenerationOptions{PackageName: "org.example.reset"}
	spec := paritySettings(want)
	spec["name"], spec["input"], spec["output_dir"], spec["input_format"] = "catalog", "source.json", "configured", "auto"
	spec["deny_warnings"], spec["allow_warnings"] = false, []string{}
	put(t, cfg, []byte("config_version=1\n[defaults]\n"+parityTable(defaults)+"[[specs]]\n"+parityTable(spec)))
	verifyConfigOptionParity(t, base, input, cfg, document, core.Auto, want)
	before := snapshot(t, filepath.Join(base, "configured"))
	// Validation=false must actually clear inherited true. An annotation that
	// requires validation must reject instead of silently enabling the option.
	put(t, input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Bounded":{"properties":{"value":{"type":"string","minLength":2}}}}}}`))
	for _, command := range [][]string{{"generate", "--config", cfg, "--diagnostic-format=json"}, {"generate", "--config", cfg, "--check", "--diagnostic-format=json"}, {"validate", "--config", cfg, "--diagnostic-format=json"}} {
		fails(t, invoke(t, t.TempDir(), command...), 1, "E_SCHEMA_UNSUPPORTED")
		if !reflect.DeepEqual(before, snapshot(t, filepath.Join(base, "configured"))) {
			t.Fatal("cleared validation failure changed owned output")
		}
	}
}
