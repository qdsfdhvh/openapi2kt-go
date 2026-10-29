package cli_test

import (
	"bytes"
	"reflect"
	"testing"
)

func TestBoundedEnumCollectionRootsNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Tree":{"type":"array","items":{"additionalProperties":{"enum":["one","two","x"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"}},"default":[{"root":"one"}]},"Alias":{"$ref":"#/components/schemas/Tree","default":[{"alias":"two"}]},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":[{"field":"one"}]}},"default":{"value":[{"outer":"two"}]}}}}}`)
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	for _, tc := range []struct{ from, to, code, pointer string }{
		{`"root":"one"`, `"root":"x"`, "E_SCHEMA_INVALID", "/components/schemas/Tree/default/0/root"},
		{`"alias":"two"`, `"alias":"future"`, "E_SCHEMA_INVALID", "/components/schemas/Alias/default/0/alias"},
		{`"field":"one"`, `"field":"x"`, "E_SCHEMA_INVALID", "/components/schemas/Use/properties/value/default/0/field"},
		{`"outer":"two"`, `"outer":"x"`, "E_SCHEMA_INVALID", "/components/schemas/Use/default/value/0/outer"},
		{`"maxLength":6`, `"maxLength":1`, "E_SCHEMA_INVALID", "/components/schemas/Tree/items/additionalProperties/maxLength"},
		{`"pattern":"^[a-z]+$"`, `"pattern":"(?=x)"`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Tree/items/additionalProperties/pattern"},
	} {
		put(t, p.Input, bytes.Replace(doc, []byte(tc.from), []byte(tc.to), 1))
		for _, flags := range [][]string{{"--validation"}, {"--validation", "--check"}} {
			got := fails(t, p.generate(t, flags...), 1, tc.code)
			if got.Diagnostics[0].Pointer == nil || *got.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("bounded collection root lost Pointer or replaced old output", tc, got)
			}
		}
	}
	put(t, p.Input, doc)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("unchecked root failure replaced old output")
	}
	result(t, p.generate(t, "--validation", "--check"), 0)
}
