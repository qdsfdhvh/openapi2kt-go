package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAllOfDTOPropertyCountOriginalNative(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "allof-dto-property-counts-qualified-envelopes.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	text := read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "Record.kt"))
	for _, token := range []string{"generatedSerializer()", "element.size < 2", "element.size > 2"} {
		if !bytes.Contains(text, []byte(token)) {
			t.Fatal(token, string(text))
		}
	}
	result(t, p.generate(t, "--validation", "--check"), 0)
	result(t, p.generate(t, "--validation"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op changed output")
	}
}

func TestAllOfDTOPropertyCountOwnershipAndExcluded(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "allof-dto-property-counts-qualified-envelopes.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	for _, mutation := range []struct{ old, next, pointer string }{
		{`"minProperties": 0`, `"minProperties": -0`, "/components/schemas/Zero/allOf/0/minProperties"},
		{`"maxProperties": 0`, `"maxProperties": 1.0`, "/components/schemas/Envelope/properties/nullable/allOf/1/maxProperties"},
		{`"maxProperties": 2`, `"maxProperties": -1`, "/components/schemas/Record/maxProperties"},
	} {
		bad := bytes.Replace(input, []byte(mutation.old), []byte(mutation.next), 1)
		if bytes.Equal(bad, input) {
			t.Fatal("mutation failed")
		}
		put(t, p.Input, bad)
		for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "CountsApi"}} {
			r := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
			if len(r.Files) != 0 || r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != mutation.pointer {
				t.Fatal(r)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("failure modified manifest/old/unowned/mtime")
			}
		}
	}
	put(t, p.Input, input)
	r := fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if *r.Diagnostics[0].Pointer != "/components/schemas/Envelope/properties/nullable/allOf/0/minProperties" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal(r)
	}
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	result(t, p.generate(t, "--validation", "--check"), 0)
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	root := doc["paths"].(map[string]any)["/patch"].(map[string]any)["patch"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	payload := root["properties"].(map[string]any)["payload"].(map[string]any)
	payload["maxProperties"] = -1
	invalidNested, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	put(t, p.Input, invalidNested)
	for _, args := range [][]string{{"--validation", "--exclude-api", "CountsApi"}, {"--validation", "--exclude-api", "CountsApi", "--check"}} {
		r := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != "/paths/~1patch/patch/requestBody/content/application~1json/schema/properties/payload/maxProperties" || len(r.Files) != 0 {
			t.Fatal(r)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("excluded invalid operation mutated output")
		}
	}
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation", "--exclude-api", "CountsApi"), 0)
	if string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal("unowned removed")
	}
	// Invalid nested counted request is validated even when every API is excluded.
	bad := bytes.Replace(input, []byte(`"minProperties": 2`), []byte(`"minProperties": -1`), -1)
	put(t, p.Input, bad)
	fails(t, p.generate(t, "--validation", "--exclude-api", "CountsApi"), 1, "E_SCHEMA_INVALID")
}

func TestAllOfDTOPropertyCountNullableComponentNative(t *testing.T) {
	p := newProject(t)
	for _, tc := range []struct{ name, bounds, at string }{
		{"Unused", `"minProperties":0`, "/components/schemas/Unused/allOf/0/minProperties"},
		{"Un~used/slot", `"minProperties":-1`, "/components/schemas/Un~0used~1slot/allOf/0/minProperties"},
		{"Unused", `"maxProperties":"bad"`, "/components/schemas/Unused/allOf/0/maxProperties"},
		{"Unused", `"minProperties":3,"maxProperties":2`, "/components/schemas/Unused/allOf/0/minProperties"},
	} {
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"` + tc.name + `":{"allOf":[{"nullable":true,` + tc.bounds + `,"properties":{"v":{"type":"string"}}}]}}}}`)
		put(t, p.Input, input)
		r := fails(t, p.generate(t, "--validation"), 1, "E_SCHEMA_UNSUPPORTED")
		if len(r.Files) != 0 || r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != tc.at {
			t.Fatal(r)
		}
	}
}
