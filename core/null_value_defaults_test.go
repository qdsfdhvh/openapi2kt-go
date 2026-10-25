package core_test

import (
	"encoding/json"
	"testing"

	"openapi-kt/core"
)

func TestJsonNullObjectMembersAndCollectionDefaults(t *testing.T) {
	for _, field := range []string{
		`{"properties":{"id":{"type":"string"},"value":{"type":"null"}},"required":["id","value"],"default":{"id":"a","value":null}}`,
		`{"properties":{"id":{"type":"string"},"value":{"enum":[null]}},"required":["id","value"],"default":{"id":"a","value":null}}`,
		`{"type":"array","items":{"type":"null"},"default":[null]}`,
		`{"type":"array","items":{"enum":[null]},"default":[null]}`,
		`{"type":"object","additionalProperties":{"type":"array","items":{"type":"null","enum":[null]}},"default":{"one":[null]}}`,
	} {
		input := schemas(`{"Holder":{"properties":{"value":` + field + `}}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				t.Fatal("originaltyped JsonNull default rejected", field, checked, e)
			}
		}
	}
}

func TestJsonNullDefaultFormerNullableMapEnumBoundary(t *testing.T) {
	field := `{"additionalProperties":{"additionalProperties":{"enum":[null],"nullable":true}},"default":{}}`
	for _, checked := range []bool{false, true} {
		rq := request(fieldDoc(field), "Dto")
		rq.Options.Validation = checked
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("exact former nullableMap nullenum default", checked, e)
		}
	}
}

func TestJsonNullDefaultWrongValuesAndVocabularyPointers(t *testing.T) {
	for _, tc := range []struct{ field, tail string }{
		{`{"type":"array","items":{"type":"null"},"default":["null"]}`, `/default/0`},
		{`{"type":"object","additionalProperties":{"enum":[null]},"default":{"a/~":"null"}}`, `/default/a~1~0`},
		{`{"properties":{"fixed":{"enum":[null]}},"default":{"fixed":"null"}}`, `/default/fixed`},
		{`{"type":"array","items":{"enum":[null,null]},"default":[]}`, `/items/enum/1`},
		{`{"type":"array","items":{"enum":[null],"default":null},"default":[]}`, `/items/default`},
		{`{"type":"array","items":{"type":"null","format":"uuid"},"default":[]}`, `/items/format`},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(fieldDoc(tc.field), "Dto")
			rq.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(rq)
			ds, ok := e.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || len(r.Artifacts.Files) != 0 {
				if ok && len(ds) > 0 && ds[0].Pointer != nil {
					t.Fatal("null default original failure Pointer", tc.field, checked, e, *ds[0].Pointer)
				}
				t.Fatal("null default original failure", tc.field, checked, e)
			}
		}
	}
}

func TestJsonNullObjectOwnDefaultsRequiredAndDomains(t *testing.T) {
	input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"requiredNull":{"type":"null"},"optionalNull":{"enum":[null],"default":null}},"required":["id","requiredNull"]},"Holder":{"properties":{"item":{"$ref":"#/components/schemas/Item","default":{"id":"a","requiredNull":null}}}}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("ordinary nullable/default/required null field roles", checked, e)
		}
	}
	for _, field := range []string{
		`{"type":"array","items":{"type":"null"},"uniqueItems":true,"default":[null]}`,
		`{"type":"array","items":{"const":null},"default":[null]}`,
		`{"type":"array","items":{"$ref":"#/components/schemas/Null"},"default":[null]}`,
	} {
		rq := request(schemas(`{"Null":{"type":"null"},"Holder":{"properties":{"value":`+field+`}}}`), "Dto")
		rq.Options.Validation = true
		if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("unadmitted null role inferred", field, e)
		}
	}
}

func TestJsonNullDefaultVocabularyLimitPrecedesEmptyValue(t *testing.T) {
	for _, size := range []int{240, 241} {
		values := make([]any, size)
		field := map[string]any{"type": "array", "items": map[string]any{"enum": values}, "default": []any{}}
		input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": map[string]any{"value": field}}}}})
		if e != nil {
			t.Fatal(e)
		}
		rq := request(input, "Dto")
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		pointer := "/components/schemas/Holder/properties/value/items/enum/1"
		code := "E_SCHEMA_INVALID"
		if size == 241 {
			pointer = "/components/schemas/Holder/properties/value/items/enum"
			code = "E_ENUM_LIMIT"
		}
		if !ok || len(ds) != 1 || ds[0].Code != code || ds[0].Pointer == nil || *ds[0].Pointer != pointer || len(r.Artifacts.Files) != 0 {
			t.Fatal("empty default bypassed vocabulary/raw limit", size, e)
		}
	}
}

func TestJsonNullDefaultRootFactoryRoles(t *testing.T) {
	for _, definitions := range []string{
		`{"Rows":{"type":"array","items":{"type":"null"},"default":[null]}}`,
		`{"Rows":{"type":"array","items":{"enum":[null],"nullable":true},"default":[null]}}`,
		`{"Rows":{"type":"object","additionalProperties":{"type":"array","items":{"type":"null"}},"default":{"one":[null]}}}`,
		`{"Rows":{"type":"array","items":{"type":"null"},"default":[null]},"Alias":{"$ref":"#/components/schemas/Rows","default":[null,null]}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(definitions), "Dto")
			rq.Options.Validation = checked
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				t.Fatal("original JsonNull collection factory", definitions, checked, e)
			}
		}
	}
	for _, definition := range []string{`{"type":"null","default":null}`, `{"enum":[null],"default":null}`} {
		rq := request(schemas(`{"Null":`+definition+`}`), "Dto")
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("former nullscalar root default boundary", definition, e)
		}
	}
	for _, tc := range []struct{ definitions, pointer string }{
		{`{"Rows":{"type":"array","items":{"type":"null"},"default":["null"]}}`, "/components/schemas/Rows/default/0"},
		{`{"Rows":{"type":"array","items":{"type":"null"}},"Alias":{"$ref":"#/components/schemas/Rows","default":[false]}}`, "/components/schemas/Alias/default/0"},
		{`{"Rows":{"type":"object","additionalProperties":{"enum":[null]},"default":{"a/~":false}}}`, "/components/schemas/Rows/default/a~1~0"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(tc.definitions), "Dto")
			rq.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(rq)
			ds, ok := e.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || len(r.Artifacts.Files) != 0 {
				t.Fatal("root factory original Pointer", tc.definitions, checked, e)
			}
		}
	}
}
