package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectionUniqueFailureAndOwnership(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "collection-unique.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	for _, tc := range []struct{ old, next, code, pointer string }{
		{`"uniqueItems": true`, `"uniqueItems": "true"`, "E_SCHEMA_INVALID", "/components/schemas/Bag/additionalProperties/uniqueItems"},
	} {
		old, next := []byte(tc.old), []byte(tc.next)
		old = bytes.ReplaceAll(old, []byte(`\n`), []byte("\n"))
		next = bytes.ReplaceAll(next, []byte(`\n`), []byte("\n"))
		bad := bytes.Replace(input, old, next, 1)
		if bytes.Equal(bad, input) {
			t.Fatal("mutation did not change source", tc)
		}
		put(t, p.Input, bad)
		for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "UniqueApi"}} {
			r := result(t, p.generate(t, args...), 1)
			if len(r.Files) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != tc.code || r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != tc.pointer {
				t.Fatal(r)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid uniqueness changed output")
			}
		}
	}
	put(t, p.Input, input)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("missing opt-in changed output")
	}
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	result(t, p.generate(t, "--validation", "--check"), 0)
	result(t, p.generate(t, "--validation", "--exclude-api", "UniqueApi"), 0)
	if string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal("unowned file changed")
	}
	text := read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "Record.kt"))
	if !bytes.Contains(text, []byte("uniqueItems")) {
		t.Fatal("lost model checks")
	}
	before = snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--exclude-api", "UniqueApi", "--check"), 0)
	result(t, p.generate(t, "--validation", "--exclude-api", "UniqueApi"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("noop touched output")
	}
}

func TestCollectionUniqueFalseCLI(t *testing.T) {
	p := newProject(t)
	plain := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"type":"object","properties":{"values":{"type":"array","items":{"type":"number"}}}}}}}`)
	noop := bytes.Replace(plain, []byte(`"type":"array"`), []byte(`"type":"array","uniqueItems":false`), 1)
	put(t, p.Input, plain)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	put(t, p.Input, noop)
	result(t, p.generate(t, "--validation", "--check"), 0)
	result(t, p.generate(t, "--validation"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("false changed output/metadata")
	}
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("false missing opt-in changed output")
	}
}

func TestCollectionUniqueExcludedOperationStillValidated(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "collection-unique.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	schema := doc["paths"].(map[string]any)["/values"].(map[string]any)["post"].(map[string]any)["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	schema["items"] = map[string]any{"type": "number"}
	// The exact formerly rejected floating response now retains its check.
	graduated, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	accepted := newProject(t)
	put(t, accepted.Input, graduated)
	result(t, accepted.generate(t, "--validation"), 0)
	result(t, accepted.generate(t, "--validation", "--check"), 0)
	result(t, accepted.generate(t, "--validation", "--exclude-api", "UniqueApi"), 0)
	result(t, accepted.generate(t, "--validation", "--exclude-api", "UniqueApi", "--check"), 0)
	// Structured elements still cannot lose an unsupported uniqueness check.
	schema["items"] = map[string]any{"type": "array", "items": map[string]any{"type": "number"}}
	bad, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	put(t, p.Input, bad)
	for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "UniqueApi"}} {
		r := result(t, p.generate(t, args...), 1)
		if len(r.Files) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "E_SCHEMA_UNSUPPORTED" || r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != "/paths/~1values/post/responses/200/content/application~1json/schema/uniqueItems" {
			t.Fatal(r)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid excluded operation changed output")
		}
	}
}
