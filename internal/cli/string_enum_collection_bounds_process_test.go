package cli_test

import (
	"bytes"
	"reflect"
	"testing"
)

func TestBoundedEnumObjectDefaultsNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"properties":{"state":{"type":"string","format":"date","enum":["one","two","x"],"minLength":2,"default":"one"}},"required":["state"],"default":{"state":"one"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":{"state":"two"}},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Value"},"default":[{"state":"two"}]}}}}}}`)
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	for _, tc := range []struct{ from, to, code, pointer string }{
		{`"default":{"state":"one"}`, `"default":{"state":"x"}`, "E_SCHEMA_INVALID", "/components/schemas/Value/default/state"},
		{`"default":{"state":"two"}`, `"default":{"state":"x"}`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default/state"},
		{`"default":[{"state":"two"}]`, `"default":[{"state":"future"}]`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/rows/default/0/state"},
		{`"minLength":2`, `"minLength":-1`, "E_SCHEMA_INVALID", "/components/schemas/Value/properties/state/minLength"},
	} {
		bad := bytes.Replace(doc, []byte(tc.from), []byte(tc.to), 1)
		if bytes.Equal(bad, doc) {
			t.Fatal("native original annotation mutation did not apply")
		}
		put(t, p.Input, bad)
		for _, flags := range [][]string{{"--validation"}, {"--validation", "--check"}} {
			got := fails(t, p.generate(t, flags...), 1, tc.code)
			if got.Diagnostics[0].Pointer == nil || *got.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("bounded object annotation changed old output or lost original Pointer", tc, got)
			}
		}
	}
	put(t, p.Input, doc)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("validation-off failure replaced existing object-default output")
	}
	result(t, p.generate(t, "--validation", "--check"), 0)
}

func TestStringEnumCollectionBoundsNativeAdmissionAndPreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"codes":{"type":"array","items":{"enum":["one","x"],"minLength":2,"maxLength":4,"pattern":"^[a-z]+$"},"default":["one"]},"labels":{"additionalProperties":{"enum":["one","x"],"minLength":2},"default":{"a/~":"one"}},"state":{"type":"string","format":"date","enum":["one","x"],"minLength":2,"default":"one"}}}}}}`)
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	for _, tc := range []struct{ old, replacement, code, pointer string }{
		{`"default":["one"]`, `"default":["x"]`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/codes/default/0"},
		{`"a/~":"one"`, `"a/~":"x"`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/labels/default/a~1~0"},
		{`"pattern":"^[a-z]+$"`, `"pattern":"(?=x)"`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/properties/codes/items/pattern"},
		{`"maxLength":4`, `"maxLength":1`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/codes/items/maxLength"},
		{`"default":"one"`, `"default":"x"`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/state/default"},
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
