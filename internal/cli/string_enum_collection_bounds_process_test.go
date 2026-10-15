package cli_test

import (
	"bytes"
	"reflect"
	"testing"
)

func TestStringEnumCollectionBoundsNativeAdmissionAndPreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"codes":{"type":"array","items":{"enum":["one","x"],"minLength":2,"maxLength":4,"pattern":"^[a-z]+$"},"default":["one"]},"labels":{"additionalProperties":{"enum":["one","x"],"minLength":2},"default":{"a/~":"one"}}}}}}}`)
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	for _, tc := range []struct{ old, replacement, code, pointer string }{
		{`"default":["one"]`, `"default":["x"]`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/codes/default/0"},
		{`"a/~":"one"`, `"a/~":"x"`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/labels/default/a~1~0"},
		{`"pattern":"^[a-z]+$"`, `"pattern":"(?=x)"`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/properties/codes/items/pattern"},
		{`"maxLength":4`, `"maxLength":1`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/codes/items/maxLength"},
	} {
		put(t, p.Input, bytes.Replace(doc, []byte(tc.old), []byte(tc.replacement), 1))
		got := fails(t, p.generate(t, "--validation"), 1, tc.code)
		if *got.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal(tc, got)
		}
	}
	put(t, p.Input, doc)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Failure changed owned output")
	}
	result(t, p.generate(t, "--validation", "--check"), 0)
}
