package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDictionaryReportsAndFailurePreserveOwnedOutput(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "model-semantics", "dictionaries.json")))
	generated := result(t, p.generate(t), 0)
	warnings := 0
	for _, warning := range generated.Warnings {
		if warning.Code == "W_ADDITIONAL_PROPERTIES_APPROXIMATION" {
			warnings++
		}
	}
	if warnings != 2 {
		t.Fatal(generated.Warnings)
	}
	before := snapshot(t, p.Output)
	for _, out := range []invocation{
		p.generate(t, "--check"),
		invoke(t, "", "validate", "--input", p.Input, "--package-name", "org.example.catalog", "--diagnostic-format", "json"),
	} {
		if got := result(t, out, 0); !reflect.DeepEqual(got.Warnings, generated.Warnings) {
			t.Fatal(got.Warnings, generated.Warnings)
		}
	}
	for _, tc := range []struct{ input, code, pointer string }{
		{`{"openapi":"3.1.0","components":{"schemas":{"Bag":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Bag"}}}}}`, "E_REF_CYCLE", "/components/schemas/Bag"},
		{`{"openapi":"3.1.0","components":{"schemas":{"Bag":{"type":"object","additionalProperties":null}}}}`, "E_SCHEMA_INVALID", "/components/schemas/Bag/additionalProperties"},
	} {
		put(t, p.Input, []byte(tc.input))
		for _, check := range []bool{false, true} {
			var flags []string
			if check {
				flags = []string{"--check"}
			}
			r := fails(t, p.generate(t, flags...), 1, tc.code)
			if r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != tc.pointer {
				t.Fatal(r)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("dictionary rejection modified owned output")
			}
		}
	}
}
