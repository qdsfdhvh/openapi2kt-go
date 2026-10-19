package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestPrimitiveMapDefaultsOriginalValuesAndGrammar(t *testing.T) {
	for _, tc := range []struct{ schema, code, tail string }{
		{`{"type":"object","additionalProperties":{"type":"string"},"default":[]}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"object","additionalProperties":{"type":"string"},"default":null}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"object","additionalProperties":{"type":"string"},"default":{"a/~":null}}`, "E_SCHEMA_INVALID", "/default/a~1~0"},
		{`{"type":"object","additionalProperties":{"type":"integer"},"default":{"a/~":2147483648}}`, "E_SCHEMA_INVALID", "/default/a~1~0"},
		{`{"type":"object","additionalProperties":{"type":"integer"},"default":{"a":1.0}}`, "E_SCHEMA_INVALID", "/default/a"},
		{`{"type":"object","additionalProperties":{"type":"number","format":"float"},"default":{"a":1e300}}`, "E_SCHEMA_INVALID", "/default/a"},
		{`{"type":"object","additionalProperties":{"type":"boolean"},"default":{"a":"true"}}`, "E_SCHEMA_INVALID", "/default/a"},
		{`{"type":"object","additionalProperties":{"type":"string","default":"x"},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "/additionalProperties/default"},
		{`{"type":"object","additionalProperties":{"type":"string"},"default":{},"nullable":"bad"}`, "E_SCHEMA_INVALID", "/nullable"},
		{`{"type":"object","additionalProperties":{"type":"string"},"required":["x"],"default":{}}`, "E_SCHEMA_INVALID", "/required"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.schema), "Dto")
			r.Options.Validation = validation
			report, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != tc.code || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal(tc, validation, err)
			}
		}
	}
}

func TestPrimitiveMapDefaultsOriginalNamedFieldGraduation(t *testing.T) {
	// Preserve the exact formerly rejected input as an executable positive case.
	input := `{"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Bag","default":{}}}},"Bag":{"type":"object","additionalProperties":{"type":"integer"}}}`
	for _, checked := range []bool{false, true} {
		r := request(schemas(input), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "HolderDto"), "public val value: org.example.models.models.BagDto? =") {
			t.Fatal("original named Map default graduation", err)
		}
	}
}

func TestPrimitiveMapDefaultsRetainChecksAndExcludedRoles(t *testing.T) {
	for _, tc := range []struct{ field, tail string }{
		{`"additionalProperties":{"type":"integer"},"minProperties":1,"default":{}`, "/default"},
		{`"additionalProperties":{"type":"integer"},"maxProperties":1,"default":{"a":1,"b":2}`, "/default"},
		{`"additionalProperties":{"type":"integer","minimum":2},"default":{"a/~":1}`, "/default/a~1~0"},
		{`"additionalProperties":{"type":"integer","multipleOf":2},"default":{"a":3}`, "/default/a"},
		{`"additionalProperties":{"type":"string","minLength":2},"default":{"a":"😀"}`, "/default/a"},
		{`"additionalProperties":{"type":"string","pattern":"^[a-z]+$"},"default":{"a":"BAD"}`, "/default/a"},
		{`"additionalProperties":{"type":"number","format":"float","exclusiveMinimum":1},"default":{"a":1.000000001}`, "/default/a"},
		{`"additionalProperties":{"type":"integer","minimum":"bad"},"default":{}`, "/additionalProperties/minimum"},
		{`"additionalProperties":{"type":"integer"},"minProperties":2,"maxProperties":1,"default":{}`, "/maxProperties"},
	} {
		r := request(fieldDoc(`{"type":"object",`+tc.field+`}`), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		if err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal(tc, err)
		}
		r.Options.Validation = false
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("checks admitted without validation", tc)
		}
	}
	mapSchema := `{"type":"object","additionalProperties":{"type":"integer"},"default":{"x":1}}`
	for _, input := range []string{
		`{"Holder":{"properties":{"value":{"type":"array","items":` + mapSchema + `}}}}`,
		`{"Holder":{"properties":{"value":{"type":"object","additionalProperties":` + mapSchema + `}}}}`,
		`{"Holder":{"properties":{"value":{"allOf":[` + mapSchema + `]}}}}`,
		`{"Holder":{"properties":{"value":{"type":"object","properties":{"fixed":{"type":"integer"}},"additionalProperties":{"type":"integer"},"default":{}}}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":` + mapSchema + `}},{"properties":{"tag":{"const":"two"}}}]}}`,
		`{"Holder":{"allOf":[{"properties":{"value":` + strings.Replace(mapSchema, `"x":1`, `"x":"bad"`, 1) + `}},{"properties":{"value":{"type":"string"}}}]}}`,
		`{"Holder":{"allOf":[{"properties":{"value":{"type":"object","additionalProperties":{"type":"integer"},"minProperties":1,"default":{"x":1}}}},{"properties":{"value":{"type":"string"}}}]}}`,
	} {
		r := request(schemas(input), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("default permission or discarded checks leaked", input, err)
		}
	}
}
