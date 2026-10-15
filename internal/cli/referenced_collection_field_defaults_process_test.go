package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReferencedCollectionFieldDefaultsNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := func(value string) []byte {
		return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":` + value + `}}},"Rows":{"type":"array","items":{"additionalProperties":{"type":"integer","format":"int64"}}}}}}`)
	}
	put(t, p.Input, doc(`[{"a":1}]`))
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--check"), 0)
	put(t, p.Input, doc(`[{"a/~":false}]`))
	failure := fails(t, p.generate(t), 1, "E_SCHEMA_INVALID")
	if *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/value/default/0/a~1~0" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal(failure)
	}
	original := doc(`[{"a":1}]`)
	for _, tc := range []struct{ old, replacement, code, pointer string }{
		{`"format":"int64"`, `"format":"int64","default":1`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Rows/items/additionalProperties/default"},
		{`"Rows":{"type":"array"`, `"Rows":{"default":[],"type":"array"`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Rows/default"},
		{`"$ref":"#/components/schemas/Rows"`, `"$ref":"#/components/schemas/Missing"`, "E_REF_MISSING", "/components/schemas/Holder/properties/value"},
	} {
		put(t, p.Input, bytes.Replace(original, []byte(tc.old), []byte(tc.replacement), 1))
		failed := fails(t, p.generate(t), 1, tc.code)
		if *failed.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("original source/membership failure changed output", tc, failed)
		}
	}
	warning := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Rows":{"type":"array","items":{"additionalProperties":{"type":"integer","format":"int64"}}},"Holder":{"allOf":[{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":[{"a":1}]}}},{}]}}}}`)
	put(t, p.Input, warning)
	denied := result(t, p.generate(t, "--deny-warnings"), 1)
	if len(denied.Files) != 0 || len(denied.Diagnostics) != 1 || denied.Diagnostics[0].Code != "E_WARNING_DENIED" || len(denied.Warnings) != 1 || denied.Warnings[0].Code != "W_ALL_OF_APPROXIMATION" {
		t.Fatal(denied)
	}
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("warning denial changed output")
	}
}
