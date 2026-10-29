package cli_test

import (
	"bytes"
	"reflect"
	"testing"
)

func TestBorrowedCheckedStringEnumNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"word":{"type":"string","format":"date","enum":["one","two","x"],"minLength":2,"maxLength":6,"default":"one"},"tree":{"type":"array","items":{"additionalProperties":{"enum":["one","two","x"],"minLength":2}}}}},"Word":{"$ref":"#/components/schemas/Owner/properties/word","default":"two"},"Maybe":{"$ref":"#/components/schemas/Word","nullable":true,"default":null},"Tree":{"$ref":"#/components/schemas/Owner/properties/tree"},"Use":{"properties":{"word":{"$ref":"#/components/schemas/Word","default":"one"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Word"},"default":["two"]},"tree":{"$ref":"#/components/schemas/Tree","default":[{"field":"one"}]}},"default":{"word":"two","rows":["one"],"tree":[{"outer":"two"}]}}}}}`)
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	for _, tc := range []struct{ from, to, code, pointer string }{
		{`"default":"one"`, `"default":"x"`, "E_SCHEMA_INVALID", "/components/schemas/Owner/properties/word/default"},
		{`"default":"two"`, `"default":"future"`, "E_SCHEMA_INVALID", "/components/schemas/Word/default"},
		{`"field":"one"`, `"field":"x"`, "E_SCHEMA_INVALID", "/components/schemas/Use/properties/tree/default/0/field"},
		{`"outer":"two"`, `"outer":"x"`, "E_SCHEMA_INVALID", "/components/schemas/Use/default/tree/0/outer"},
		{`"maxLength":6`, `"maxLength":1`, "E_SCHEMA_INVALID", "/components/schemas/Owner/properties/word/maxLength"},
	} {
		bad := bytes.Replace(doc, []byte(tc.from), []byte(tc.to), 1)
		put(t, p.Input, bad)
		for _, flags := range [][]string{{"--validation"}, {"--validation", "--check"}} {
			got := fails(t, p.generate(t, flags...), 1, tc.code)
			if got.Diagnostics[0].Pointer == nil || *got.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("checked borrowed source changed output or diagnostic Pointer", tc, got)
			}
		}
	}
	put(t, p.Input, doc)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("unchecked borrowing failure changed old output")
	}
	result(t, p.generate(t, "--validation", "--check"), 0)
}
