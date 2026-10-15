package cli_test

import (
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestHTTPExclusionsRemoveOwnedOperationModels(t *testing.T) {
	for _, fixture := range []string{"operation-models.json", "request-presence.json", "deep-object.json", "deep-bindings.json", "projections.json"} {
		t.Run(fixture, func(t *testing.T) { testHTTPExclusionOwnedModels(t, fixture) })
	}
}

func testHTTPExclusionOwnedModels(t *testing.T, fixture string) {
	t.Helper()
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", fixture)))
	result(t, p.generate(t), 0)
	unowned := filepath.Join(p.Output, "keep.txt")
	put(t, unowned, []byte("keep"))
	r := result(t, p.generate(t, "--exclude-api", "DefaultApi"), 0)
	expected := map[string]int{"operation-models.json": 3, "request-presence.json": 4, "deep-object.json": 2, "deep-bindings.json": 1, "projections.json": 2}[fixture]
	if len(r.Files) != expected || slices.ContainsFunc(r.Files, func(s string) bool { return strings.Contains(s, "/api/") }) {
		t.Fatal(r)
	}
	if string(read(t, unowned)) != "keep" {
		t.Fatal("unowned file changed")
	}
	result(t, p.generate(t, "--exclude-api", "DefaultApi", "--check"), 0)
	for path := range snapshot(t, p.Output) {
		if strings.Contains(filepath.ToSlash(path), "/api/") || strings.HasSuffix(filepath.ToSlash(path), "/models/Patch.kt") {
			t.Fatal("stale owned operation file", path)
		}
	}
}

func TestHTTPClientOptions(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "basic.json")))
	flags := []string{"--api-client-name", "ChosenClient", "--exclude-api", "WidgetsApi", "--exclude-api", "WidgetsApi"}
	r := result(t, p.generate(t, flags...), 0)
	if len(r.Files) != 5 {
		t.Fatal(r)
	}
	if !slices.ContainsFunc(r.Files, func(s string) bool { return strings.HasSuffix(s, "/ChosenClient.kt") }) || slices.ContainsFunc(r.Files, func(s string) bool { return strings.HasSuffix(s, "/WidgetsApi.kt") }) {
		t.Fatal(r)
	}
	result(t, p.generate(t, append(slices.Clone(flags), "--check")...), 0)
	result(t, invoke(t, "", "validate", "--input", p.Input, "--diagnostic-format", "json", "--api-client-name", "ChosenClient", "--exclude-api", "WidgetsApi"), 0)
	before := snapshot(t, p.Output)
	for _, flag := range []string{"--api-client-name", "--exclude-api"} {
		for _, extra := range [][]string{nil, {"--check"}} {
			args := append([]string{flag, "../Escape"}, extra...)
			fails(t, p.generate(t, args...), 1, "E_NAME_INVALID")
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid client options changed output")
			}
		}
	}
}
