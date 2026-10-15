package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNativeEnumConsumersNativeDefaultOffAndCheckedFailurePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/native-enum-consumers.json")))
	result(t, p.generate(t), 0)
	unvalidated := snapshot(t, p.Output)
	result(t, p.generate(t, "--check"), 0)
	if !reflect.DeepEqual(unvalidated, snapshot(t, p.Output)) {
		t.Fatal("default-off check changed output")
	}
	report := result(t, p.generate(t, "--validation"), 0)
	found := false
	for _, name := range report.Files {
		if strings.HasSuffix(name, "DefaultApi.kt") {
			found = strings.Contains(string(read(t, filepath.Join(p.Output, filepath.FromSlash(name)))), "does not belong to enum.")
		}
	}
	if !found {
		t.Fatal("typed operation membership checks missing")
	}
	checked := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	if !reflect.DeepEqual(checked, snapshot(t, p.Output)) {
		t.Fatal("checked no-op changed files")
	}
	p.change(t, func(doc map[string]any) { schemas(doc)["Code"].(map[string]any)["enum"] = []any{1, 1} })
	fails(t, p.generate(t, "--validation"), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(checked, snapshot(t, p.Output)) {
		t.Fatal("bad catalog vocabulary changed outputs")
	}
}
