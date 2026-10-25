package cli_test

import (
	"encoding/json"
	"openapi-kt/core"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDTOAllOfNativeParityAndFailurePreservation(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/dto-allof-unique.json"))
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		p := newProject(t)
		put(t, p.Input, input)
		flags := []string{"--validation", "--model-name-suffix", "Dto", "--model-style", string(style)}
		result(t, p.generate(t, flags...), 0)
		expected, e := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: true}})
		if e != nil {
			t.Fatal(e)
		}
		for _, f := range expected.Artifacts.Files {
			if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
				t.Fatal("native/core bytes", f.Path)
			}
		}
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
		result(t, p.generate(t, flags...), 0)
		result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--validation", "--diagnostic-format=json"), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("readonly/noop output drift")
		}
		for _, tc := range []struct {
			field, tail string
			value       any
		}{{"leafRefStates", "/0", []any{"future"}}, {"leafRefNumbers", "/0", []any{2}}, {"leafRefBridge", "/0", []any{"future"}}, {"leafRefMap", "/a/0", map[string]any{"a": []any{"future"}}}, {"stringRefList", "/0", []any{"future"}}, {"stringRefMap", "/a", map[string]any{"a": "future"}}, {"stringRefBridge", "/0", []any{"future"}}, {"stringRefTree", "/a/0", map[string]any{"a": []any{"future"}}}} {
			var doc map[string]any
			if e := json.Unmarshal(input, &doc); e != nil {
				t.Fatal(e)
			}
			defs := doc["components"].(map[string]any)["schemas"].(map[string]any)
			defs["AllOfBase"].(map[string]any)["properties"].(map[string]any)[tc.field].(map[string]any)["default"] = tc.value
			data, e := json.Marshal(doc)
			if e != nil {
				t.Fatal(e)
			}
			put(t, p.Input, data)
			failed := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			pointer := "/components/schemas/AllOfBase/properties/" + tc.field + "/default" + tc.tail
			if failed.Diagnostics[0].Pointer == nil || *failed.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("strict annotation/Pointer/old output", failed)
			}
		}
	}
}
