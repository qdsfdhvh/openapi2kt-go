package cli_test

import (
	"bytes"
	"reflect"
	"testing"
)

func TestReferencedEnumCollectionFactoryNativePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Word":{"enum":["one","two","x"],"default":"one"},"WordAlias":{"$ref":"#/components/schemas/Word","default":"two"},"Tree":{"additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/WordAlias"}},"default":{"root":["one"]}},"Alias":{"$ref":"#/components/schemas/Tree","default":{"a/~":["two"]}},"Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":{"field":["one"]}}},"required":["value"]}}}}`)
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
			doc = bytes.Replace(doc, []byte(`"enum":["one","two","x"]`), []byte(`"enum":["one","two","x"],"minLength":2`), 1)
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		for _, tc := range []struct{ from, to, pointer string }{
			{`"default":"one"`, `"default":"future"`, "/components/schemas/Word/default"},
			{`"default":"two"`, `"default":"future"`, "/components/schemas/WordAlias/default"},
			{`"root":["one"]`, `"root":["future"]`, "/components/schemas/Tree/default/root/0"},
			{`"a/~":["two"]`, `"a/~":["future"]`, "/components/schemas/Alias/default/a~1~0/0"},
			{`"field":["one"]`, `"field":["future"]`, "/components/schemas/Use/properties/value/default/field/0"},
			{`"nullable":true`, `"nullable":false`, "/components/schemas/Maybe/default"},
		} {
			put(t, p.Input, bytes.Replace(doc, []byte(tc.from), []byte(tc.to), 1))
			for _, check := range []bool{false, true} {
				args := append([]string{}, flags...)
				if check {
					args = append(args, "--check")
				}
				got := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
				if got.Diagnostics[0].Pointer == nil || *got.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("reference factory failure changed old output or Pointer", tc, got)
				}
			}
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, append(flags, "--check")...), 0)
	}
}
