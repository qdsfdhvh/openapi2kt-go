package cli_test

import (
	"bytes"
	"reflect"
	"testing"
)

func TestBorrowedEnumCollectionDefaultNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"tree":{"type":"array","items":{"enum":["one","two","x"],"minLength":2},"default":["one"]}}},"Tree":{"$ref":"#/components/schemas/Owner/properties/tree","default":["two"]},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Tree","default":["one"]}}}}}}`)
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	for _, tc := range []struct{ from, to, code, pointer string }{
		{`"default":["one"]`, `"default":["x"]`, "E_SCHEMA_INVALID", "/components/schemas/Owner/properties/tree/default/0"},
		{`"default":["two"]`, `"default":["future"]`, "E_SCHEMA_INVALID", "/components/schemas/Tree/default/0"},
		{`"default":["two"]`, `"default":["x"]`, "E_SCHEMA_INVALID", "/components/schemas/Tree/default/0"},
		{`"minLength":2`, `"minLength":-1`, "E_SCHEMA_INVALID", "/components/schemas/Owner/properties/tree/items/minLength"},
	} {
		put(t, p.Input, bytes.Replace(doc, []byte(tc.from), []byte(tc.to), 1))
		for _, flags := range [][]string{{"--validation"}, {"--validation", "--check"}} {
			got := fails(t, p.generate(t, flags...), 1, tc.code)
			if got.Diagnostics[0].Pointer == nil || *got.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("collection factory failure changed old output or Pointer", tc, got)
			}
		}
	}
	put(t, p.Input, doc)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("unchecked source failure replaced old collection factory output")
	}
	result(t, p.generate(t, "--validation", "--check"), 0)
}
