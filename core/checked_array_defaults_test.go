package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestCheckedPrimitiveArrayDefaults(t *testing.T) {
	for _, field := range []string{
		`{"type":"array","minItems":1,"maxItems":3,"uniqueItems":true,"items":{"type":"integer","minimum":0,"multipleOf":2},"default":[0,2]}`,
		`{"type":"array","items":{"type":"string","minLength":1,"maxLength":3},"default":["😀"]}`,
		`{"type":"array","uniqueItems":true,"items":{"type":"string","nullable":true,"pattern":"^[a-z]+$"},"default":[null,"ok"]}`,
		`{"type":"array","items":{"type":"number","format":"float","minimum":1},"default":[1.000000001]}`,
		`{"type":"array","nullable":true,"minItems":1,"items":{"type":"integer","minimum":1},"default":null}`,
		// Original counterexamples now admitted only with validation enabled.
		`{"type":"array","items":{"type":"string","minLength":1},"default":[]}`,
		`{"type":"array","items":{"type":"string"},"default":[],"minItems":0}`,
		`{"type":"array","items":{"type":"string"},"default":[],"uniqueItems":true}`,
	} {
		t.Run(field, func(t *testing.T) {
			r := request(fieldDoc(field), "Dto")
			if _, err := core.GenerateWithDiagnostics(r); err == nil {
				t.Fatal("Validation-off admitted constraints")
			}
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Nondeterministic checked default", err)
			}
			if !strings.Contains(source(t, got, "RecordDto"), "require(") {
				t.Fatal("Lost runtime constraints")
			}
		})
	}
}

func TestCheckedPrimitiveArrayDefaultsDoNotLoseOriginalSources(t *testing.T) {
	for _, input := range []string{
		`{"Holder":{"allOf":[{"properties":{"value":{"type":"array","items":{"type":"integer","minimum":2},"default":[1]}}},{"properties":{"value":{"type":"string"}}}]}}`,
		`{"Holder":{"allOf":[{"properties":{"value":{"type":"array","items":{"type":"integer","minimum":2},"default":[2]}}},{"properties":{"value":{"type":"string"}}}]}}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","minItems":1,"items":{"type":"string"},"default":["x"]}}}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":{"type":"array","minItems":1,"items":{"type":"string"},"default":["x"]}}},{"properties":{"tag":{"const":"two"}}}]}}`,
	} {
		r := request(schemas(input), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Default permission leaked or erased original constraints", input, err)
		}
	}
}

func TestCheckedPrimitiveArrayDefaultDiagnostics(t *testing.T) {
	floatingUniqueGraduation(t, fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"number"},"default":[]}`))
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"array","minItems":2,"items":{"type":"integer"},"default":[1]}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"array","maxItems":0,"items":{"type":"integer"},"default":[1]}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"array","uniqueItems":true,"items":{"type":"integer"},"default":[0,-0]}`, "E_SCHEMA_INVALID", "/default/1"},
		{`{"type":"array","uniqueItems":true,"items":{"type":"string"},"default":["a","\u0061"]}`, "E_SCHEMA_INVALID", "/default/1"},
		{`{"type":"array","uniqueItems":true,"items":{"type":"boolean","nullable":true},"default":[null,true,null]}`, "E_SCHEMA_INVALID", "/default/2"},
		{`{"type":"array","items":{"type":"integer","minimum":2},"default":[2,1]}`, "E_SCHEMA_INVALID", "/default/1"},
		{`{"type":"array","items":{"type":"integer","multipleOf":2},"default":[3]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"string","minLength":2},"default":["😀"]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"string","pattern":"^[a-z]+$"},"default":["A"]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"string","pattern":"."},"default":["😀"]}`, "E_SCHEMA_UNSUPPORTED", "/default/0"},
		{`{"type":"array","items":{"type":"number","format":"float","exclusiveMinimum":1},"default":[1.000000001]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","minItems":2,"maxItems":1,"items":{"type":"integer"},"default":[]}`, "E_SCHEMA_INVALID", "/maxItems"},
		{`{"type":"array","minItems":-1,"items":{"type":"integer"},"default":[]}`, "E_SCHEMA_INVALID", "/minItems"},
		{`{"type":"array","items":{"type":"integer","minimum":"bad"},"default":[]}`, "E_SCHEMA_INVALID", "/items/minimum"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			r := request(fieldDoc(tc.field), "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil {
				t.Fatal("Admitted invalid checked default")
			}
			d := err.(core.Diagnostics)[0]
			if d.Code != tc.code || d.Pointer == nil || *d.Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatalf("%+v, want %s %s", d, tc.code, tc.tail)
			}
		})
	}
}
