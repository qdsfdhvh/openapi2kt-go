package cli_test

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNestedStringEnumBoundsNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"properties":{"tree":{"type":"array","items":{"additionalProperties":{"type":"string","format":"date","enum":["one","two","x"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"}},"default":[{"a/~":"one"}]}},"default":{"tree":[{"root":"two"}]}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":{"tree":[{"outer":"one"}]}}}}}}}`)
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	for _, tc := range []struct{ from, to, code, pointer string }{
		{`"a/~":"one"`, `"a/~":"x"`, "E_SCHEMA_INVALID", "/components/schemas/Value/properties/tree/default/0/a~1~0"},
		{`"root":"two"`, `"root":"future"`, "E_SCHEMA_INVALID", "/components/schemas/Value/default/tree/0/root"},
		{`"outer":"one"`, `"outer":"x"`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default/tree/0/outer"},
		{`"maxLength":6`, `"maxLength":1`, "E_SCHEMA_INVALID", "/components/schemas/Value/properties/tree/items/additionalProperties/maxLength"},
		{`"pattern":"^[a-z]+$"`, `"pattern":"(?=x)"`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Value/properties/tree/items/additionalProperties/pattern"},
	} {
		bad := bytes.Replace(doc, []byte(tc.from), []byte(tc.to), 1)
		if bytes.Equal(bad, doc) {
			t.Fatal("nested native mutation did not apply")
		}
		put(t, p.Input, bad)
		for _, flags := range [][]string{{"--validation"}, {"--validation", "--check"}} {
			got := fails(t, p.generate(t, flags...), 1, tc.code)
			if got.Diagnostics[0].Pointer == nil || *got.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("nested bounded annotation changed output or lost Pointer", tc, got)
			}
		}
	}
	put(t, p.Input, doc)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("validation-off failure changed accepted nested output")
	}
	result(t, p.generate(t, "--validation", "--check"), 0)
}
