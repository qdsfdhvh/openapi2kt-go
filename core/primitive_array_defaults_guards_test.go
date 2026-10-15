package core_test

import (
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestPrimitiveArrayDefaultsValidateOriginalMembersAndExcludedGrammar(t *testing.T) {
	for _, tc := range []struct{ schema, code, tail string }{
		{`{"type":"array","items":{"type":"string"},"default":{}}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"array","items":{"type":"string"},"default":null}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"array","items":{"type":"string"},"default":["ok",null]}`, "E_SCHEMA_INVALID", "/default/1"},
		{`{"type":"array","items":{"type":"integer"},"default":[0,2147483648]}`, "E_SCHEMA_INVALID", "/default/1"},
		{`{"type":"array","items":{"type":"integer"},"default":[1.0]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"number","format":"float"},"default":[1e300]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"boolean"},"default":["true"]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"string","default":"x"},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/default"},
		{`{"type":"array","items":{"type":"string"},"default":[],"nullable":"bad"}`, "E_SCHEMA_INVALID", "/nullable"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.schema), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != tc.code || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, validation, err)
			}
		}
	}
}

func TestPrimitiveArrayDefaultsKeepValueAndPinnedRolesExcludedAndDiscardedSourcesChecked(t *testing.T) {
	for _, input := range []string{
		`{"Array":{"type":"array","items":{"type":"string"},"default":[]}}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"type":"string"},"default":[]}}}}}`,
		`{"Holder":{"properties":{"value":{"allOf":[{"type":"array","items":{"type":"string"}}],"default":[]}}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":{"type":"array","items":{"type":"string"},"default":[]}}},{"properties":{"tag":{"const":"two"}}}]}}`,
		`{"Holder":{"allOf":[{"properties":{"value":{"type":"array","items":{"type":"integer"},"default":[1.5]}}},{"properties":{"value":{"type":"string"}}}]}}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Array default permission leaked or dropped original source", input, err)
		}
	}
}
