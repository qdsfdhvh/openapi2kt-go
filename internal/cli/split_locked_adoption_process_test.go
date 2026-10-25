package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func splitAdoptionInputs(t *testing.T, base string) (string, string) {
	t.Helper()
	source := `{"openapi":"3.1.0","components":{"schemas":{"A":{"properties":{"name":{"type":"string"}}},"B":{"properties":{"count":{"type":"integer"}}}}},"paths":{"/a":{"get":{"operationId":"alpha","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"type":"string"}},"required":["value"]}}}}}}},"/b":{"get":{"operationId":"beta","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"type":"string"}},"required":["value"]}}}}}}}}}`
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(source), &doc); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(base, "source.json")
	put(t, input, []byte(source))
	var components map[string]json.RawMessage
	_ = json.Unmarshal(doc["components"], &components)
	var models map[string]json.RawMessage
	_ = json.Unmarshal(components["schemas"], &models)
	delete(models, "B")
	components["schemas"], _ = json.Marshal(models)
	doc["components"], _ = json.Marshal(components)
	api, _ := json.Marshal(doc)
	put(t, filepath.Join(base, "api.json"), api)
	put(t, filepath.Join(base, "models.json"), []byte(`{"openapi":"3.1.0","components":{"schemas":{"B":{"properties":{"count":{"type":"integer"}}}}}}`))
	cfg := filepath.Join(base, "split.toml")
	put(t, cfg, []byte("config_version=1\n[defaults]\noutput_dir='generated'\npackage_name='org.example.split'\nmodel_name_suffix='Dto'\napi_client_name='SplitClient'\nhoist_shared_shapes=true\nlock_hoisted_names=true\n[[specs]]\nname='api'\ninput='api.json'\n[[specs]]\nname='models'\ninput='models.json'\n"))
	return input, cfg
}

func TestSplitLockedAdoptionNativeExactUnionAndReplay(t *testing.T) {
	base := t.TempDir()
	input, cfg := splitAdoptionInputs(t, base)
	root := filepath.Join(base, "generated")
	result(t, invoke(t, base, "generate", "--input", input, "--output", root, "--package-name=org.example.split", "--model-name-suffix=Dto", "--api-client-name=SplitClient", "--hoist-shared-shapes", "--lock-hoisted-names", "--diagnostic-format=json"), 0)
	before := snapshot(t, root)
	args := []string{"generate", "--config", cfg, "--adopt-locked-owner", "--diagnostic-format=json"}
	result(t, invoke(t, base, append(append([]string(nil), args...), "--check")...), 3)
	result(t, invoke(t, base, lifecycleValidate(args)...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("split adoption readonly wrote state")
	}
	result(t, invoke(t, base, args...), 0)
	after := snapshot(t, root)
	for path, old := range before {
		if filepath.Ext(path) == ".kt" && !reflect.DeepEqual(old, after[path]) {
			t.Fatal("split adoption changed Kotlin", path)
		}
	}
	var manifest struct {
		Version int
		Owners  map[string][]string
	}
	if err := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &manifest); err != nil || manifest.Version != 4 || len(manifest.Owners) != 2 || len(manifest.Owners["api"]) <= 1 || len(manifest.Owners["models"]) != 1 {
		t.Fatal("not a genuine split", manifest, err)
	}
	result(t, invoke(t, base, append(append([]string(nil), args...), "--check")...), 0)
	result(t, invoke(t, base, args...), 0)
	result(t, invoke(t, base, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
	if !reflect.DeepEqual(after, snapshot(t, root)) {
		t.Fatal("split replay/ordinary check changed metadata")
	}
}

func TestSplitLockedAdoptionNativeFailurePreservesOriginalSource(t *testing.T) {
	for _, kind := range []string{"changed", "missing", "allocation", "profile", "occupied", "partial-selection", "warning"} {
		base := t.TempDir()
		input, cfg := splitAdoptionInputs(t, base)
		root := filepath.Join(base, "generated")
		result(t, invoke(t, base, "generate", "--input", input, "--output", root, "--package-name=org.example.split", "--model-name-suffix=Dto", "--api-client-name=SplitClient", "--hoist-shared-shapes", "--lock-hoisted-names", "--diagnostic-format=json"), 0)
		args := []string{"generate", "--config", cfg, "--adopt-locked-owner", "--diagnostic-format=json"}
		code, diagnostic := 4, "E_LOCKED_OWNER_ADOPTION_CHANGED"
		switch kind {
		case "changed":
			put(t, filepath.Join(base, "models.json"), []byte(`{"openapi":"3.1.0","components":{"schemas":{"B":{"properties":{"count":{"type":"integer"},"added":{"type":"string"}}}}}}`))
		case "missing":
			put(t, filepath.Join(base, "models.json"), []byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"properties":{"name":{"type":"string"}}}}}}`))
		case "allocation":
			var doc map[string]json.RawMessage
			var paths map[string]json.RawMessage
			_ = json.Unmarshal(read(t, filepath.Join(base, "api.json")), &doc)
			_ = json.Unmarshal(doc["paths"], &paths)
			delete(paths, "/b")
			doc["paths"], _ = json.Marshal(paths)
			data, _ := json.Marshal(doc)
			put(t, filepath.Join(base, "api.json"), data)
		case "profile":
			put(t, cfg, []byte(strings.Replace(string(read(t, cfg)), "org.example.split", "org.example.changed", 1)))
			code = 1
			diagnostic = "E_NAME_LOCK_PROFILE"
		case "occupied":
			put(t, filepath.Join(root, namedLockPath("models")), []byte("foreign"))
			diagnostic = "E_OUTPUT_COLLISION"
		case "partial-selection":
			args = append(args, "--spec=api")
			diagnostic = "E_LOCKED_OWNER_ADOPTION_INVALID"
		case "warning":
			put(t, filepath.Join(base, "models.json"), []byte(`{"openapi":"3.1.0","components":{"schemas":{"B":{"properties":{"count":{"type":"integer"},"empty":{"type":"object"}}}}}}`))
			put(t, cfg, []byte(strings.Replace(string(read(t, cfg)), "[defaults]", "[defaults]\ndeny_warnings=true", 1)))
			code = 1
			diagnostic = "E_WARNING_DENIED"
		}
		before := snapshot(t, root)
		for _, command := range [][]string{args, append(append([]string(nil), args...), "--check"), lifecycleValidate(args)} {
			if kind == "warning" {
				denied := warningDenied(t, invoke(t, base, command...))
				if len(denied.Warnings) != 1 {
					t.Fatal("split adoption lost original warning", denied)
				}
			} else {
				fails(t, invoke(t, base, command...), code, diagnostic)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("invalid split lost source", kind)
			}
		}
	}
}
