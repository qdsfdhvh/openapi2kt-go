package cli_test

import (
	"bytes"
	"reflect"
	"testing"
)

func TestCheckedStringEnumRootNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Word":{"type":"string","format":"date","enum":["one","two","x"],"minLength":2,"maxLength":6,"default":"one"},"Alias":{"$ref":"#/components/schemas/Word","default":"two"},"Use":{"properties":{"word":{"$ref":"#/components/schemas/Alias"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"default":["one"]}}}}}}`)
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	for _, tc := range []struct{ from, to, pointer string }{
		{`"default":"one"`, `"default":"x"`, "/components/schemas/Word/default"},
		{`"default":"two"`, `"default":"future"`, "/components/schemas/Alias/default"},
		{`"default":["one"]`, `"default":["x"]`, "/components/schemas/Use/properties/rows/default/0"},
		{`"maxLength":6`, `"maxLength":1`, "/components/schemas/Word/maxLength"},
	} {
		put(t, p.Input, bytes.Replace(doc, []byte(tc.from), []byte(tc.to), 1))
		for _, flags := range [][]string{{"--validation"}, {"--validation", "--check"}} {
			got := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			if got.Diagnostics[0].Pointer == nil || *got.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("checked root failure changed old output or Pointer", tc, got)
			}
		}
	}
	put(t, p.Input, doc)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("unchecked enum root failure changed output")
	}
	result(t, p.generate(t, "--validation", "--check"), 0)
}
