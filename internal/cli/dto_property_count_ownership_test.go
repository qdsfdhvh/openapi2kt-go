package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDTOPropertyCountFailureAndOwnership(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "dto-property-counts.json"))
	// Start with an ordinary old manifest before opting into the new codec.
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	var strip func(any)
	strip = func(value any) {
		switch v := value.(type) {
		case map[string]any:
			delete(v, "minProperties")
			delete(v, "maxProperties")
			for _, x := range v {
				strip(x)
			}
		case []any:
			for _, x := range v {
				strip(x)
			}
		}
	}
	strip(doc)
	plain, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	put(t, p.Input, plain)
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	old := snapshot(t, p.Output)
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation", "--check"), 3)
	if !reflect.DeepEqual(old, snapshot(t, p.Output)) {
		t.Fatal("check mutated old manifest")
	}
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	text := read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "Node.kt"))
	if !bytes.Contains(text, []byte("generatedSerializer()")) {
		t.Fatal("lost nominal auto delegate")
	}
	disabled := fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if disabled.Diagnostics[0].Pointer == nil || *disabled.Diagnostics[0].Pointer != "/components/schemas/Node/minProperties" {
		t.Fatal(disabled)
	}
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("disabled validation changed output")
	}
	for _, tc := range []struct{ old, next, code, pointer string }{
		{`"minProperties": 1`, `"minProperties": -1`, "E_SCHEMA_INVALID", "/components/schemas/Record/minProperties"},
		{`"maxProperties": 3`, `"maxProperties": 0`, "E_SCHEMA_INVALID", "/components/schemas/Record/maxProperties"},
		{`"minProperties": 0`, `"minProperties": "zero"`, "E_SCHEMA_INVALID", "/components/schemas/Zero/minProperties"},
	} {
		bad := bytes.Replace(input, []byte(tc.old), []byte(tc.next), 1)
		if bytes.Equal(bad, input) {
			t.Fatal("unchanged mutation", tc)
		}
		put(t, p.Input, bad)
		for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "CountsApi"}} {
			r := fails(t, p.generate(t, args...), 1, tc.code)
			if r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != tc.pointer {
				t.Fatal(r)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("failure changed output/manifest/unowned")
			}
		}
	}
	// Excluding the only API cannot suppress invalid ordinary inline counts.
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	schema := doc["paths"].(map[string]any)["/patch"].(map[string]any)["patch"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	schema["maxProperties"] = -1
	bad, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	put(t, p.Input, bad)
	r := fails(t, p.generate(t, "--validation", "--exclude-api", "CountsApi"), 1, "E_SCHEMA_INVALID")
	if r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != "/paths/~1patch/patch/requestBody/content/application~1json/schema/maxProperties" {
		t.Fatal(r)
	}
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("excluded failure changed output")
	}
	put(t, p.Input, input)
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	result(t, p.generate(t, "--validation", "--check"), 0)
	result(t, p.generate(t, "--validation"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("noop changed mtimes/hash/manifest")
	}
	result(t, p.generate(t, "--validation", "--exclude-api", "CountsApi"), 0)
	result(t, p.generate(t, "--validation", "--exclude-api", "CountsApi", "--check"), 0)
	if string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal("unowned modified")
	}
	// Migration back to a prior unbounded manifest removes only owned runtime.
	put(t, p.Input, plain)
	result(t, p.generate(t), 0)
	restored := snapshot(t, p.Output)
	if len(old) != len(restored) {
		t.Fatal("unbounded manifest inventory not restored")
	}
	for path, previous := range old {
		if previous.Bytes != restored[path].Bytes || previous.Mode != restored[path].Mode {
			t.Fatal("unbounded bytes/manifest not restored", path)
		}
	}
	if old["keep.txt"] != restored["keep.txt"] {
		t.Fatal("unowned mtime changed")
	}
}

func TestDTOPropertyCountUnsupportedNativePointer(t *testing.T) {
	p := newProject(t)
	for _, shape := range []string{`"properties":{}`, `"properties":{"kind":{"type":"string","const":"fixed"}}`, `"allOf":[{"type":"object","properties":{"v":{"type":"string"}}}]`} {
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Unused":{"type":"object","minProperties":0,` + shape + `}}}}`)
		put(t, p.Input, input)
		if strings.HasPrefix(shape, `"allOf"`) {
			result(t, p.generate(t, "--validation"), 0)
			text := read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "Unused.kt"))
			if !bytes.Contains(text, []byte("generatedSerializer()")) || !bytes.Contains(text, []byte("element.size < 0")) {
				t.Fatal("lost effective wrapper/count", string(text))
			}
			r := fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
			if r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != "/components/schemas/Unused/minProperties" {
				t.Fatal(r)
			}
			continue
		}
		r := fails(t, p.generate(t, "--validation"), 1, "E_SCHEMA_UNSUPPORTED")
		if r.Diagnostics[0].Pointer == nil || !strings.HasSuffix(*r.Diagnostics[0].Pointer, "/minProperties") {
			t.Fatal(r)
		}
	}
}
