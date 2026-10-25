package cli_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"openapi-kt/core"
	"openapi-kt/internal/output"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func retainedAdoptionNativeFixture(t *testing.T, style string, checked, optimized bool) (string, string, string) {
	base := t.TempDir()
	input, cfg := splitAdoptionInputs(t, base)
	root := filepath.Join(base, "generated")
	direct := []string{"generate", "--input", input, "--output", root, "--package-name=org.example.split", "--model-name-suffix=Dto", "--api-client-name=SplitClient", "--hoist-shared-shapes", "--lock-hoisted-names", "--model-style", style, "--diagnostic-format=json"}
	text := string(read(t, cfg))
	text = strings.Replace(text, "[defaults]", "[defaults]\nmodel_style='"+style+"'", 1)
	if checked {
		text = strings.Replace(text, "[defaults]", "[defaults]\nvalidation=true", 1)
		direct = append(direct, "--validation")
	}
	if optimized {
		text = strings.Replace(text, "[defaults]", "[defaults]\ndeduplicate_models=true\nmerge_identical_models=true", 1)
		direct = append(direct, "--deduplicate-models", "--merge-identical-models")
	}
	put(t, cfg, []byte(text))
	result(t, invoke(t, base, direct...), 0)

	header := strings.Split(string(read(t, cfg)), "[[specs]]")[0]
	put(t, cfg, []byte(header+"[[specs]]\nname='models'\ninput='models.json'\n"))
	return base, cfg, root
}

func TestRetainedLockedAdoptionNativeProperSubsetAndReplay(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			for _, optimized := range []bool{false, true} {
				base, cfg, root := retainedAdoptionNativeFixture(t, style, checked, optimized)
				header := strings.Split(string(read(t, cfg)), "[[specs]]")[0]
				before := snapshot(t, root)
				var old struct{ Files map[string]string }
				if e := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &old); e != nil {
					t.Fatal(e)
				}
				fails(t, invoke(t, base, "generate", "--config", cfg, "--adopt-locked-owner", "--check", "--diagnostic-format=json"), 4, "E_LOCKED_OWNER_ADOPTION_CHANGED")
				args := []string{"generate", "--config", cfg, "--adopt-locked-owner", "--retain-locked-owner=original", "--diagnostic-format=json"}
				result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 3)
				result(t, invoke(t, base, lifecycleValidate(args)...), 0)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("partial retained adoption readonly wrote")
				}
				result(t, invoke(t, base, args...), 0)
				after := snapshot(t, root)
				for path, entry := range before {
					if filepath.Ext(path) == ".kt" && !reflect.DeepEqual(entry, after[path]) {
						t.Fatal("retained source Kotlin/attrs changed", path)
					}
				}
				var next struct {
					Version int
					Owners  map[string][]string
					Locks   map[string]string `json:"name_locks"`
				}
				if e := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &next); e != nil {
					t.Fatal(e)
				}
				if next.Version != 4 || len(next.Owners) != 2 || len(next.Owners["original"]) != len(old.Files) || len(next.Owners["models"]) >= len(old.Files) || len(next.Locks) != 2 {
					t.Fatal("not retained full original plus genuine subset", next)
				}
				result(t, invoke(t, base, "generate", "--config", cfg, "--spec=models", "--check", "--diagnostic-format=json"), 0)
				result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 0)
				result(t, invoke(t, base, args...), 0)
				put(t, cfg, []byte(header+"[[specs]]\nname='original'\ninput='source.json'\n[[specs]]\nname='models'\ninput='models.json'\n"))
				result(t, invoke(t, base, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
				if !reflect.DeepEqual(after, snapshot(t, root)) {
					t.Fatal("ordinary full/partial config or action replay rewrote metadata")
				}

			}
		}
	}
}

func TestRetainedLockedAdoptionNativeProjectedSingletonAndHistoricalSource(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "generated")
	full := read(t, filepath.Join(repo, "tests/fixtures/http/split-adoption-source.json"))
	options := core.GenerationOptions{PackageName: "org.example.retained", ModelNameSuffix: "Dto", APIClientName: "SplitClient", Validation: true, ModelStyle: core.PlainClass, HoistSharedShapes: true, LockHoistedNames: true}
	request := core.GenerationRequest{Document: full, Format: core.JSON, Options: options}
	first, e := core.Generate(request)
	if e != nil {
		t.Fatal(e)
	}
	var doc, paths map[string]json.RawMessage
	if e := json.Unmarshal(full, &doc); e != nil {
		t.Fatal(e)
	}
	_ = json.Unmarshal(doc["paths"], &paths)
	delete(paths, "/b")
	doc["paths"], _ = json.Marshal(paths)
	current, _ := json.Marshal(doc)
	request.Document = current
	request.PreviousNameLock = first.NameLock
	request.ProjectPreviousNameLock = true
	original, e := core.Generate(request)
	if e != nil {
		t.Fatal(e)
	}
	// Persist actual existing API projection output, not handcrafted locks/code.
	if _, e := output.WriteNameLocked(root, original, nil, false); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(base, "source.json"), current)
	var components, schemas map[string]json.RawMessage
	_ = json.Unmarshal(doc["components"], &components)
	_ = json.Unmarshal(components["schemas"], &schemas)
	delete(schemas, "B")
	components["schemas"], _ = json.Marshal(schemas)
	doc["components"], _ = json.Marshal(components)
	subset, _ := json.Marshal(doc)
	put(t, filepath.Join(base, "part.json"), subset)
	cfg := filepath.Join(base, "retained.toml")
	header := "config_version=1\n[defaults]\noutput_dir='generated'\npackage_name='org.example.retained'\nmodel_name_suffix='Dto'\napi_client_name='SplitClient'\nmodel_style='plain_class'\nvalidation=true\nhoist_shared_shapes=true\nlock_hoisted_names=true\n"
	put(t, cfg, []byte(header+"[[specs]]\nname='part'\ninput='part.json'\n"))
	before := snapshot(t, root)
	raw := read(t, filepath.Join(root, ".openapi-kt-names.json"))
	args := []string{"generate", "--config", cfg, "--adopt-locked-owner", "--retain-locked-owner=original", "--diagnostic-format=json"}
	result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 3)
	result(t, invoke(t, base, args...), 0)
	after := snapshot(t, root)
	for path, entry := range before {
		if filepath.Ext(path) == ".kt" && !reflect.DeepEqual(entry, after[path]) {
			t.Fatal("singleton/historical code changed")
		}
	}
	if !reflect.DeepEqual(raw, read(t, filepath.Join(root, namedLockPath("original")))) {
		t.Fatal("original singleton raw lock changed")
	}
	var lock struct {
		Entries []struct{ Occurrences []string }
	}
	if e := json.Unmarshal(read(t, filepath.Join(root, namedLockPath("part"))), &lock); e != nil {
		t.Fatal(e)
	}
	if len(lock.Entries) == 0 {
		t.Fatal("candidate lost original retained singleton")
	}
	for _, entry := range lock.Entries {
		if len(entry.Occurrences) != 1 {
			t.Fatal("fixture is not singleton projection")
		}
	}
	result(t, invoke(t, base, "generate", "--config", cfg, "--spec=part", "--check", "--diagnostic-format=json"), 0)
	result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 0)
	if !reflect.DeepEqual(after, snapshot(t, root)) {
		t.Fatal("singleton replay rewrote state")
	}
	if _, e := os.Stat(filepath.Join(root, ".openapi-kt-names.json")); !os.IsNotExist(e) {
		t.Fatal("old owned direct slot not removed")
	}
}

func TestRetainedLockedAdoptionNativeUsageReportAndPreInputProfile(t *testing.T) {
	for _, args := range [][]string{
		{"generate", "--retain-locked-owner=original"},
		{"generate", "--config=missing.toml", "--retain-locked-owner=original"},
		{"generate", "--config=missing.toml", "--adopt-locked-owner", "--retain-locked-owner="},
		{"generate", "--config=missing.toml", "--adopt-locked-owner", "--retain-locked-owner=../invalid"},
		{"generate", "--config=missing.toml", "--adopt-locked-owner", "--retain-locked-owner=original", "--retain-locked-owner=other"},
		{"generate", "--config=missing.toml", "--adopt-locked-owner", "--retain-locked-owner=original", "--spec=part"},
		{"generate", "--config=missing.toml", "--adopt-locked-owner", "--retain-locked-owner=original", "--adopt-locked-owner-from=old", "--spec=part"},
	} {
		fails(t, invoke(t, "", append(args, "--diagnostic-format=json")...), 2, "E_USAGE")
	}
	for _, tail := range [][]string{{"--reconcile"}, {"--adopt-legacy"}, {"--rename-owner=old:new"}, {"--name-lock-action=reset"}, {"--move-output-from=old", "--output=new"}, {"--reassign-locked-owners"}} {
		args := append([]string{"generate", "--config=missing.toml", "--adopt-locked-owner", "--retain-locked-owner=original", "--diagnostic-format=json"}, tail...)
		fails(t, invoke(t, "", args...), 2, "E_USAGE")
	}
	help := invoke(t, "", "generate", "--help")
	if help.Code != 0 || !strings.Contains(string(help.Stdout), "--retain-locked-owner") {
		t.Fatal("retain action missing help")
	}
	base, cfg, root := retainedAdoptionNativeFixture(t, "plain_class", true, true)
	before := snapshot(t, root)
	out := invoke(t, base, "generate", "--config", cfg, "--adopt-locked-owner", "--retain-locked-owner=original", "--check", "--diagnostic-format=json")
	result(t, out, 3)
	var data struct {
		Migration struct {
			Owner         string                          `json:"retained_owner"`
			Before        int                             `json:"before_version"`
			After         int                             `json:"after_version"`
			RetainedFiles []struct{ Path, SHA256 string } `json:"retained_files"`
			Candidates    []struct{ Owner string }
			Lock          struct {
				Before  string `json:"before_sha256"`
				After   string `json:"after_sha256"`
				Symbols []json.RawMessage
			} `json:"retained_lock"`
		} `json:"retained_locked_adoption"`
	}
	if e := json.Unmarshal(out.Stdout, &data); e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(read(t, filepath.Join(root, ".openapi-kt-names.json")))
	if data.Migration.Lock.Before != hex.EncodeToString(sum[:]) || data.Migration.Lock.After != data.Migration.Lock.Before {
		t.Fatal("report does not bind actual original raw lock")
	}
	if data.Migration.Owner != "original" || data.Migration.Before != 3 || data.Migration.After != 4 || len(data.Migration.RetainedFiles) < 2 || len(data.Migration.Candidates) != 1 || len(data.Migration.Lock.Symbols) == 0 || strings.Contains(string(out.Stdout), base) {
		t.Fatal("invalid retained report", string(out.Stdout))
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("retained report wrote")
	}
	text := string(read(t, cfg))
	header := strings.Split(text, "[[specs]]")[0]
	put(t, cfg, []byte(header+"[[specs]]\nname='first'\ninput='missing.json'\n[[specs]]\nname='second'\ninput='missing-again.json'\npackage_name='org.example.changed'\n"))
	fails(t, invoke(t, base, "generate", "--config", cfg, "--adopt-locked-owner", "--retain-locked-owner=original", "--diagnostic-format=json"), 1, "E_NAME_LOCK_PROFILE")
	put(t, cfg, []byte(strings.Replace(text, "name='models'", "name='original'", 1)))
	fails(t, invoke(t, base, "generate", "--config", cfg, "--adopt-locked-owner", "--retain-locked-owner=original", "--diagnostic-format=json"), 4, "E_RETAINED_LOCKED_ADOPTION_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("invalid profile/retainer identity wrote")
	}
}

func TestRetainedLockedAdoptionNativeMulticandidateAndWarning(t *testing.T) {
	for _, deny := range []bool{false, true} {
		base, cfg, root := retainedAdoptionNativeFixture(t, "data_class", true, false)
		text := string(read(t, cfg))
		header := strings.Split(text, "[[specs]]")[0]
		text = header + "[[specs]]\nname='models'\ninput='models.json'\n[[specs]]\nname='api'\ninput='api.json'\n"
		if deny {
			text = strings.Replace(text, "[defaults]", "[defaults]\ndeny_warnings=true", 1)
			var doc, components, models, model, properties map[string]json.RawMessage
			input := filepath.Join(base, "models.json")
			_ = json.Unmarshal(read(t, input), &doc)
			_ = json.Unmarshal(doc["components"], &components)
			_ = json.Unmarshal(components["schemas"], &models)
			_ = json.Unmarshal(models["B"], &model)
			_ = json.Unmarshal(model["properties"], &properties)
			properties["empty"] = json.RawMessage(`{"type":"object"}`)
			model["properties"], _ = json.Marshal(properties)
			models["B"], _ = json.Marshal(model)
			components["schemas"], _ = json.Marshal(models)
			doc["components"], _ = json.Marshal(components)
			bytes, _ := json.Marshal(doc)
			put(t, input, bytes)
		}
		put(t, cfg, []byte(text))
		before := snapshot(t, root)
		args := []string{"generate", "--config", cfg, "--adopt-locked-owner", "--retain-locked-owner=original", "--diagnostic-format=json"}
		if deny {
			warningDenied(t, invoke(t, base, args...))
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("warning policy wrote")
			}
			continue
		}
		result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 3)
		result(t, invoke(t, base, args...), 0)
		after := snapshot(t, root)
		for path, entry := range before {
			if filepath.Ext(path) == ".kt" && !reflect.DeepEqual(entry, after[path]) {
				t.Fatal("multi-candidate adoption changed code")
			}
		}
		result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 0)
		// A full ordinary inventory includes the retained original source spec.
		put(t, cfg, []byte(text+"[[specs]]\nname='original'\ninput='source.json'\n"))
		result(t, invoke(t, base, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
		if !reflect.DeepEqual(after, snapshot(t, root)) {
			t.Fatal("multi-candidate replay changed raw metadata")
		}
	}
}
