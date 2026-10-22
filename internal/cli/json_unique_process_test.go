package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestJSONUniqueNativePreservesOutputOnInvalidOriginalDefault(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/json-unique.json")))
	args := []string{"--model-name-suffix=Dto", "--validation"}
	result(t, p.generate(t, args...), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, append(append([]string{}, args...), "--check")...), 0)
	result(t, p.generate(t, args...), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("default check/no-op touched files")
	}
	p.change(t, func(d map[string]any) {
		schemas(d)["Record"].(map[string]any)["properties"].(map[string]any)["values"].(map[string]any)["uniqueItems"] = "invalid"
	})
	for _, checked := range []bool{false, true} {
		invocation := append([]string{}, args...)
		if checked {
			invocation = append(invocation, "--check")
		}
		report := fails(t, p.generate(t, invocation...), 1, "E_SCHEMA_INVALID")
		if *report.Diagnostics[0].Pointer != "/components/schemas/Record/properties/values/uniqueItems" {
			t.Fatal(report)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid default changed old output")
		}
	}
}

func TestJSONUniqueNativePreservesOutputOnExcludedReferenceSource(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/json-unique.json")))
	args := []string{"--model-name-suffix=Dto", "--validation"}
	result(t, p.generate(t, args...), 0)
	before := snapshot(t, p.Output)
	p.change(t, func(d map[string]any) {
		schemas(d)["Raw"] = map[string]any{"allOf": []any{map[string]any{"type": "object"}, map[string]any{"properties": map[string]any{}}}}
	})
	// The exact former composed-raw rejection is now an accepted source.
	result(t, p.generate(t, args...), 0)
	result(t, p.generate(t, append(append([]string{}, args...), "--check")...), 0)
	before = snapshot(t, p.Output)
	p.change(t, func(d map[string]any) {
		schemas(d)["Raw"] = map[string]any{"allOf": []any{map[string]any{"properties": map[string]any{"kind": map[string]any{"const": "only"}}}, map[string]any{"type": "object"}}}
	})
	for _, check := range []bool{false, true} {
		invocation := append([]string{}, args...)
		if check {
			invocation = append(invocation, "--check")
		}
		report := fails(t, p.generate(t, invocation...), 1, "E_SCHEMA_UNSUPPORTED")
		if *report.Diagnostics[0].Pointer != "/components/schemas/Record/properties/raw/uniqueItems" {
			t.Fatal(report)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("excluded referenced JSON equality changed old output")
		}
	}
}

func TestJSONUniqueNativePreservesOutputOnInvalidCoverDescendant(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/json-unique.json")))
	args := []string{"--model-name-suffix=Dto", "--validation"}
	result(t, p.generate(t, args...), 0)
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned note"))
	before := snapshot(t, p.Output)
	warningDenied(t, p.generate(t, append(append([]string{}, args...), "--deny-warnings")...))
	result(t, p.generate(t, append(append([]string{}, args...), "--check")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("cover warning denial/check changed old output")
	}
	p.change(t, func(d map[string]any) {
		schemas(d)["Peer"].(map[string]any)["oneOf"].([]any)[1].(map[string]any)["minLength"] = 1
	})
	for _, checked := range []bool{false, true} {
		invocation := append([]string{}, args...)
		if checked {
			invocation = append(invocation, "--check")
		}
		report := fails(t, p.generate(t, invocation...), 1, "E_SCHEMA_UNSUPPORTED")
		if *report.Diagnostics[0].Pointer != "/components/schemas/Peer/oneOf/1/minLength" {
			t.Fatal(report)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("bad complete cover descendant changed old output")
		}
	}
}
