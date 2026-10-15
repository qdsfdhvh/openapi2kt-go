package cli_test

import (
	"bytes"
	"encoding/json"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func lockInput(t testing.TB, root string) string {
	t.Helper()
	path := filepath.Join(root, "input.json")
	put(t, path, read(t, filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")))
	return path
}
func lockGrow(t testing.TB, input string) {
	t.Helper()
	var doc, paths, alpha, post map[string]json.RawMessage
	if e := json.Unmarshal(read(t, input), &doc); e != nil {
		t.Fatal(e)
	}
	_ = json.Unmarshal(doc["paths"], &paths)
	_ = json.Unmarshal(paths["/alpha"], &alpha)
	_ = json.Unmarshal(alpha["post"], &post)
	post["operationId"] = json.RawMessage(`"aardvark"`)
	alpha["post"], _ = json.Marshal(post)
	paths["/aardvark"], _ = json.Marshal(alpha)
	doc["paths"], _ = json.Marshal(paths)
	data, _ := json.Marshal(doc)
	put(t, input, data)
}
func lockArgs(input, root string) []string {
	return []string{"generate", "--input", input, "--output", root, "--package-name=org.example.dedup", "--model-name-suffix=Dto", "--validation", "--hoist-shared-shapes", "--deduplicate-models", "--lock-hoisted-names", "--diagnostic-format=json"}
}
func lockCore(t testing.TB, input, root string, previous []byte, style core.ModelStyle, qualified bool) core.GenerationArtifacts {
	t.Helper()
	a, e := core.Generate(core.GenerationRequest{Document: read(t, input), Format: core.Auto, PreviousNameLock: previous, Options: core.GenerationOptions{PackageName: "org.example.dedup", ModelNameSuffix: "Dto", Validation: true, HoistSharedShapes: true, DeduplicateModels: true, LockHoistedNames: true, ModelStyle: style, QualifyHoistedNames: qualified}})
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range a.Files {
		if !bytes.Equal(read(t, filepath.Join(root, f.Path)), []byte(f.Content)) {
			t.Fatal("native output differs from actual core", f.Path)
		}
	}
	if !bytes.Equal(read(t, filepath.Join(root, ".openapi-kt-names.json")), a.NameLock) {
		t.Fatal("native state differs from actual core")
	}
	return a
}
func TestNameLockCLINativeCoreStatePipeline(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, qualified := range []bool{false, true} {
			base := t.TempDir()
			input := lockInput(t, base)
			root := filepath.Join(base, "output")
			args := append(lockArgs(input, root), "--model-style="+string(style))
			if qualified {
				args = append(args, "--qualify-hoisted-names")
			}
			result(t, invoke(t, repo, append(args, "--check")...), 3)
			if _, e := os.Stat(root); !os.IsNotExist(e) {
				t.Fatal("freshcheck created root")
			}
			v := append([]string(nil), args...)
			v[0] = "validate"
			result(t, invoke(t, repo, v...), 0)
			if _, e := os.Stat(root); !os.IsNotExist(e) {
				t.Fatal("freshvalidate created root")
			}
			report := result(t, invoke(t, repo, args...), 0)
			for _, p := range report.Files {
				if !strings.HasSuffix(p, ".kt") {
					t.Fatal("metadata leaked in files report")
				}
			}
			original := lockCore(t, input, root, nil, style, qualified)
			before := snapshot(t, root)
			for _, extra := range [][]string{{"--check"}, nil} {
				result(t, invoke(t, repo, append(args, extra...)...), 0)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("noop/check wrote state")
				}
			}
			result(t, invoke(t, repo, v...), 0)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("validate mutated")
			}
			lockGrow(t, input)
			result(t, invoke(t, repo, append(args, "--check")...), 3)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("growth check wrote")
			}
			result(t, invoke(t, repo, args...), 0)
			next := lockCore(t, input, root, original.NameLock, style, qualified)
			if bytes.Equal(original.NameLock, next.NameLock) {
				t.Fatal("new occurrence not persisted")
			}
			after := snapshot(t, root)
			result(t, invoke(t, repo, args...), 0)
			if !reflect.DeepEqual(after, snapshot(t, root)) {
				t.Fatal("grown replay changed bytes/modes/mtime")
			}
		}
	}
}
func TestNameLockCLIFlagsDiagnosticsAndFailurePreservation(t *testing.T) {
	base := t.TempDir()
	input := lockInput(t, base)
	root := filepath.Join(base, "output")
	args := lockArgs(input, root)
	for _, tail := range [][]string{{"--lock-hoisted-names=FALSE"}, {"--lock-hoisted-names=yes"}, {"--lock-hoisted-names=true", "--lock-hoisted-names"}, {"--lock-hoisted-names", "true"}} {
		fails(t, invoke(t, repo, append(deduplicateArgs(root), tail...)...), 2, "E_USAGE")
	}
	fails(t, invoke(t, repo, append(deduplicateArgs(root), "--lock-hoisted-names")...), 2, "E_USAGE")
	for _, command := range []string{"generate", "validate"} {
		h := invoke(t, repo, command, "--help")
		if h.Code != 0 || !strings.Contains(string(h.Stdout), "--lock-hoisted-names[=true|false]") {
			t.Fatal("missinghelp", h)
		}
	}
	for _, tail := range [][]string{{"--output", root}, {"--model-name-suffix=Dto"}, {"--hoist-shared-shapes", "--lock-hoisted-names"}} {
		fails(t, invoke(t, repo, append([]string{"validate", "--input", input, "--diagnostic-format=json"}, tail...)...), 2, "E_USAGE")
	}
	fails(t, invoke(t, repo, "download", "--url", "https://example.test/schema", "--output", input, "--lock-hoisted-names", "--diagnostic-format=json"), 2, "E_USAGE")
	result(t, invoke(t, repo, args...), 0)
	before := snapshot(t, root)
	// Existing locked roots cannot implicitly disable or change the profile.
	off := append([]string(nil), args...)
	for i, a := range off {
		if a == "--lock-hoisted-names" {
			off[i] = "--lock-hoisted-names=false"
		}
	}
	fails(t, invoke(t, repo, off...), 4, "E_MANIFEST_VERSION")
	altered := append([]string(nil), args...)
	for i, a := range altered {
		if a == "--model-name-suffix=Dto" {
			altered[i] = "--model-name-suffix=Other"
		}
	}
	fails(t, invoke(t, repo, altered...), 1, "E_NAME_LOCK_PROFILE")
	for _, extra := range [][]string{nil, {"--check"}} {
		bad := append(args, extra...)
		state := filepath.Join(root, ".openapi-kt-names.json")
		old := read(t, state)
		put(t, state, []byte("synthetic edited metadata"))
		edited := snapshot(t, root)
		fails(t, invoke(t, repo, bad...), 4, "E_NAME_LOCK_MODIFIED")
		if !reflect.DeepEqual(edited, snapshot(t, root)) {
			t.Fatal("editedstate failure wrote")
		}
		put(t, state, old)
	}
	if len(before) != len(snapshot(t, root)) {
		t.Fatal("failure added/deleted paths")
	}
	// Preserve constructor-property order while deleting an original occurrence.
	var doc, paths map[string]json.RawMessage
	_ = json.Unmarshal(read(t, input), &doc)
	_ = json.Unmarshal(doc["paths"], &paths)
	delete(paths, "/beta")
	doc["paths"], _ = json.Marshal(paths)
	data, _ := json.Marshal(doc)
	put(t, input, data)
	final := snapshot(t, root)
	fails(t, invoke(t, repo, args...), 1, "E_NAME_LOCK_REMOVED")
	if !reflect.DeepEqual(final, snapshot(t, root)) {
		t.Fatal("corefailure wrote")
	}
}
