package core_test

import (
	"testing"

	"openapi-kt/core"
)

func TestDTOScalarReferenceGetterUnique(t *testing.T) {
	for _, definition := range []string{`{"Value":{"type":"integer"}}`, `{"Value":{"type":"string","format":"date-time"}}`, `{"Value":{"$ref":"#/components/schemas/Hops/properties/leaf"},"Hops":{"properties":{"leaf":{"type":"number","format":"float"}}}}`} {
		rq := request(schemas(definition[:len(definition)-1]+`,"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("original scalar reference getter", definition, e)
		}
	}
}

func TestDTOScalarReferenceOwnDefaultsAndMappedKeys(t *testing.T) {
	for _, tc := range []struct{ definition, field, values string }{
		{`{"Value":{"type":"number","format":"float"}}`, `{"$ref":"#/components/schemas/Value","default":0}`, `[{}, {"value":-0.0}]`},
		{`{"Value":{"type":"string","format":"uuid"}}`, `{"$ref":"#/components/schemas/Value","default":"550E8400-E29B-41D4-A716-446655440000"}`, `[{}, {"value":"550e8400-e29b-41d4-a716-446655440000"}]`},
		{`{"Value":{"type":"string","format":"date-time"},"Alias":{"$ref":"#/components/schemas/Value"}}`, `{"$ref":"#/components/schemas/Alias","default":"2024-01-01T00:00:00Z"}`, `[{}, {"value":"2024-01-01T01:00:00+01:00"}]`},
	} {
		rq := request(schemas(tc.definition[:len(tc.definition)-1]+`,"Item":{"properties":{"value":`+tc.field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":`+tc.values+`}}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			if ok && len(ds) > 0 && ds[0].Pointer != nil {
				t.Fatal("reference initializer equality pointer", tc.field, e, *ds[0].Pointer)
			}
			t.Fatal("reference initializer equality", tc.field, e)
		}
	}
}

func TestDTOScalarReferenceNullableDefaultAndRequiredContexts(t *testing.T) {
	for _, field := range []string{`{"$ref":"#/components/schemas/Alias","nullable":false,"default":0}`, `{"$ref":"#/components/schemas/Alias","default":null}`} {
		for _, required := range []string{`[]`, `["value"]`} {
			rq := request(schemas(`{"Value":{"type":"integer","nullable":true},"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Item":{"properties":{"value":`+field+`},"required":`+required+`},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
			rq.Options.Validation = true
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				t.Fatal("reference nullable/default/required", field, required, e)
			}
		}
	}
}

func TestDTOScalarReferenceOriginalDefaultFailures(t *testing.T) {
	for _, tc := range []struct{ defs, field, tail string }{
		{`{"Value":{"type":"integer","minimum":1}}`, `{"$ref":"#/components/schemas/Value","default":0}`, "/default"},
		{`{"Value":{"type":"integer","nullable":true}}`, `{"$ref":"#/components/schemas/Value","nullable":false,"default":null}`, "/default"},
		{`{"Value":{"type":"string","format":"date-time"}}`, `{"$ref":"#/components/schemas/Value","default":"bad"}`, "/default"},
	} {
		rq := request(schemas(tc.defs[:len(tc.defs)-1]+`,"Item":{"properties":{"value":`+tc.field+`},"required":["value"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/value"+tc.tail || len(r.Artifacts.Files) != 0 {
			t.Fatal("reference required/default failure", tc.field, e)
		}
	}
}

func TestDTOScalarReferenceGraphFailuresRemainBounded(t *testing.T) {
	for _, definition := range []string{
		`{"Value":{"$ref":"#/components/schemas/Alias"},"Alias":{"$ref":"#/components/schemas/Value"}}`,
		`{"Value":{"$ref":"#/components/schemas/Missing"}}`,
		`{"Value":{"type":"integer","const":0}}`,
	} {
		rq := request(schemas(definition[:len(definition)-1]+`,"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("unproved reference target role", definition, e)
		}
	}
}
