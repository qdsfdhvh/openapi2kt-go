package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func reassignmentNativeFixture(t *testing.T, style string, checked, optimized bool) (string, string, string) {
	base := t.TempDir()
	input, cfg := splitAdoptionInputs(t, base)
	root := filepath.Join(base, "generated")
	put(t, cfg, []byte(strings.Replace(string(read(t, cfg)), "[defaults]", "[defaults]\nqualify_hoisted_names=true", 1)))
	direct := []string{"generate", "--input", input, "--output", root, "--package-name=org.example.split", "--model-name-suffix=Dto", "--api-client-name=SplitClient", "--hoist-shared-shapes", "--qualify-hoisted-names", "--lock-hoisted-names", "--diagnostic-format=json"}
	configText := string(read(t, cfg))
	configText = strings.Replace(configText, "[defaults]", "[defaults]\nmodel_style='"+style+"'", 1)
	direct = append(direct, "--model-style", style)
	if checked {
		configText = strings.Replace(configText, "[defaults]", "[defaults]\nvalidation=true", 1)
		direct = append(direct, "--validation")
	}
	if optimized {
		configText = strings.Replace(configText, "[defaults]", "[defaults]\ndeduplicate_models=true\nmerge_identical_models=true", 1)
		direct = append(direct, "--deduplicate-models", "--merge-identical-models")
	}
	put(t, cfg, []byte(configText))
	result(t, invoke(t, base, direct...), 0)
	var doc, paths, operation, method map[string]json.RawMessage
	if e := json.Unmarshal(read(t, input), &doc); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(doc["paths"], &paths); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(paths["/a"], &operation); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(operation["get"], &method); e != nil {
		t.Fatal(e)
	}
	method["operationId"] = json.RawMessage(`"aardvark"`)
	operation["get"], _ = json.Marshal(method)
	paths["/0"], _ = json.Marshal(operation)
	doc["paths"], _ = json.Marshal(paths)
	data, _ := json.Marshal(doc)
	put(t, input, data)
	result(t, invoke(t, base, direct...), 0) // Keep historical AlphaResponseDto despite a new earlier peer.
	var api map[string]json.RawMessage
	if e := json.Unmarshal(read(t, filepath.Join(base, "api.json")), &api); e != nil {
		t.Fatal(e)
	}
	api["paths"] = doc["paths"]
	data, _ = json.Marshal(api)
	put(t, filepath.Join(base, "api.json"), data)
	result(t, invoke(t, base, "generate", "--config", cfg, "--adopt-locked-owner", "--diagnostic-format=json"), 0)
	// Transfer complete API/client/hoisted memberships to the existing models
	// owner; api retains A, models retains B. Global original Kotlin is unchanged.
	var models map[string]json.RawMessage
	if e := json.Unmarshal(read(t, filepath.Join(base, "models.json")), &models); e != nil {
		t.Fatal(e)
	}
	models["paths"] = api["paths"]
	delete(api, "paths")
	data, _ = json.Marshal(api)
	put(t, filepath.Join(base, "api.json"), data)
	data, _ = json.Marshal(models)
	put(t, filepath.Join(base, "models.json"), data)
	fails(t, invoke(t, base, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 1, "E_NAME_LOCK_REMOVED")
	return base, cfg, root
}

func TestLockedOwnerReassignmentNativeHistoricalNamesMetadataOnly(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			for _, optimized := range []bool{false, true} {
				base, cfg, root := reassignmentNativeFixture(t, style, checked, optimized)
				before := snapshot(t, root)
				args := []string{"generate", "--config", cfg, "--reassign-locked-owners", "--diagnostic-format=json"}
				result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 3)
				result(t, invoke(t, base, lifecycleValidate(args)...), 0)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("reassignment readonly wrote")
				}
				result(t, invoke(t, base, args...), 0)
				after := snapshot(t, root)
				for path, old := range before {
					if filepath.Ext(path) == ".kt" && !reflect.DeepEqual(old, after[path]) {
						t.Fatal("historical Kotlin/public names or attrs changed", path)
					}
				}
				result(t, invoke(t, base, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
				result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 0)
				result(t, invoke(t, base, args...), 0)
				if !reflect.DeepEqual(after, snapshot(t, root)) {
					t.Fatal("reassignment replay rewrote metadata")
				}

			}
		}
	}
}

func TestLockedOwnerReassignmentNativeUsageAndReport(t *testing.T) {
	for _, args := range [][]string{
		{"generate", "--reassign-locked-owners"},
		{"generate", "--input=-", "--reassign-locked-owners"},
		{"generate", "--config=missing.toml", "--spec=api", "--reassign-locked-owners"},
		{"generate", "--config=missing.toml", "--reassign-locked-owners=true"},
		{"generate", "--config=missing.toml", "--reassign-locked-owners", "--reassign-locked-owners"},
	} {
		fails(t, invoke(t, "", append(args, "--diagnostic-format=json")...), 2, "E_USAGE")
	}
	for _, tail := range [][]string{{"--reconcile"}, {"--adopt-locked-owner"}, {"--adopt-legacy", "--spec=api"}, {"--adopt-locked-owner-from=api", "--adopt-locked-owner", "--spec=models"}, {"--name-lock-action=reset"}, {"--rename-owner=api:newapi"}, {"--move-output-from=old", "--output=new"}} {
		args := append([]string{"generate", "--config=missing.toml", "--reassign-locked-owners"}, tail...)
		fails(t, invoke(t, "", append(args, "--diagnostic-format=json")...), 2, "E_USAGE")
	}
	help := invoke(t, "", "generate", "--help")
	if help.Code != 0 || !strings.Contains(string(help.Stdout), "--reassign-locked-owners") {
		t.Fatal("new action missing help")
	}
	base, cfg, root := reassignmentNativeFixture(t, "data_class", true, true)
	before := snapshot(t, root)
	out := invoke(t, base, "generate", "--config", cfg, "--reassign-locked-owners", "--check", "--diagnostic-format=json")
	result(t, out, 3)
	var raw struct {
		Migration struct {
			BeforeHash string   `json:"manifest_before_sha256"`
			AfterHash  string   `json:"manifest_after_sha256"`
			Before     []string `json:"before_owners"`
			After      []string `json:"after_owners"`
			Files      []struct{ Path, SHA256 string }
			Owners     []struct {
				Owner       string
				BeforeFiles []string `json:"before_files"`
				AfterFiles  []string `json:"after_files"`
			}
			Symbols []struct{ Classifier, Descriptor string } `json:"source_symbols"`
		} `json:"locked_owner_reassignment"`
	}
	if e := json.Unmarshal(out.Stdout, &raw); e != nil {
		t.Fatal(e)
	}
	if len(raw.Migration.BeforeHash) != 64 || len(raw.Migration.AfterHash) != 64 || raw.Migration.BeforeHash == raw.Migration.AfterHash || !reflect.DeepEqual(raw.Migration.Before, []string{"api", "models"}) || !reflect.DeepEqual(raw.Migration.After, raw.Migration.Before) || len(raw.Migration.Symbols) == 0 || len(raw.Migration.Owners) != 2 || strings.Contains(string(out.Stdout), base) {
		t.Fatal("incomplete or unsafe report", string(out.Stdout))
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("report mutated output")
	}
}

func TestLockedOwnerReassignmentNativeNewAndRemovedOwnerIDs(t *testing.T) {
	for _, mode := range []string{"new-id", "single-owner"} {
		base, cfg, root := reassignmentNativeFixture(t, "plain_class", true, true)
		before := snapshot(t, root)
		text := string(read(t, cfg))
		if mode == "new-id" {
			text = strings.Replace(text, "name='api'", "name='newapi'", 1)
			text = strings.Replace(text, "name=\"api\"", "name=\"newapi\"", 1)
		} else {
			text = strings.Split(text, "[[specs]]")[0] + "[[specs]]\nname='models'\ninput='source.json'\n"
		}
		put(t, cfg, []byte(text))
		args := []string{"generate", "--config", cfg, "--reassign-locked-owners", "--diagnostic-format=json"}
		result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 3)
		result(t, invoke(t, base, args...), 0)
		after := snapshot(t, root)
		for path, old := range before {
			if filepath.Ext(path) == ".kt" && !reflect.DeepEqual(old, after[path]) {
				t.Fatal("owner addition/removal changed Kotlin", mode, path)
			}
		}
		var manifest struct {
			Owners map[string][]string
			Locks  map[string]string `json:"name_locks"`
		}
		if e := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &manifest); e != nil {
			t.Fatal(e)
		}
		if manifest.Owners["api"] != nil || len(manifest.Locks) != len(manifest.Owners) || manifest.Owners["models"] == nil {
			t.Fatal("old owner not removed/new inventory incomplete")
		}
		result(t, invoke(t, base, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
		result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 0)
		if !reflect.DeepEqual(after, snapshot(t, root)) {
			t.Fatal("new owner inventory replay changed state")
		}
	}
}

func TestLockedOwnerReassignmentNativeWarningPolicyBeforeMetadata(t *testing.T) {
	base, cfg, root := reassignmentNativeFixture(t, "data_class", true, false)
	var doc, components, schemas, model, properties map[string]json.RawMessage
	input := filepath.Join(base, "api.json")
	if e := json.Unmarshal(read(t, input), &doc); e != nil {
		t.Fatal(e)
	}
	_ = json.Unmarshal(doc["components"], &components)
	_ = json.Unmarshal(components["schemas"], &schemas)
	_ = json.Unmarshal(schemas["A"], &model)
	_ = json.Unmarshal(model["properties"], &properties)
	properties["unused"] = json.RawMessage(`{"type":"object"}`)
	model["properties"], _ = json.Marshal(properties)
	schemas["A"], _ = json.Marshal(model)
	components["schemas"], _ = json.Marshal(schemas)
	doc["components"], _ = json.Marshal(components)
	data, _ := json.Marshal(doc)
	put(t, input, data)
	put(t, cfg, []byte(strings.Replace(string(read(t, cfg)), "[defaults]", "[defaults]\ndeny_warnings=true", 1)))
	before := snapshot(t, root)
	warningDenied(t, invoke(t, base, "generate", "--config", cfg, "--reassign-locked-owners", "--diagnostic-format=json"))
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("warning policy changed ownership")
	}
}
