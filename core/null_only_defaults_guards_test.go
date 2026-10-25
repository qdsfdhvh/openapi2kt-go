package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullOnlyDefaultsKeepRequiredNullableAndRequestPatch(t *testing.T) {
	for _, value := range []string{`{"type":"null","nullable":true,"default":null}`, `{"type":"null","enum":[null],"nullable":true,"default":null}`, `{"enum":[null],"nullable":true,"default":null}`} {
		for _, validation := range []bool{false, true} {
			r := request(schemas(`{"Holder":{"properties":{"required":`+value+`,"optional":`+value+`},"required":["required"]}}`), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(value, validation, err)
			}
			text := source(t, got, "HolderDto")
			if !strings.Contains(text, "public val required: kotlinx.serialization.json.JsonNull?,") || strings.Contains(text, "public val required: kotlinx.serialization.json.JsonNull? =") || !strings.Contains(text, "public val optional: kotlinx.serialization.json.JsonNull? = null") {
				t.Fatal("A required nullable default replaced the constructor argument", text)
			}
			input := []byte(`{"openapi":"3.1.0","paths":{"/patch":{"patch":{"operationId":"patch","requestBody":{"required":true,"content":{"application/json":{"schema":{"properties":{"required":` + value + `,"optional":` + value + `},"required":["required"]}}}},"responses":{"204":{}}}}}}`)
			r.Document = input
			got, err = core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(value, validation, err)
			}
			text = source(t, got, "PatchInputDto")
			if !strings.Contains(text, "public val required: kotlinx.serialization.json.JsonNull?,") || !strings.Contains(text, "public val optional: org.example.models.models.Patch<kotlinx.serialization.json.JsonNull> = org.example.models.models.Patch.Absent") {
				t.Fatal("A null default annotation became a request update", text)
			}
		}
	}
}

func TestNullOnlyDefaultOriginalGrammarAndUnrepresentableValues(t *testing.T) {
	for _, tc := range []struct{ value, code, tail string }{
		{`{"type":"null","default":"null"}`, "E_SCHEMA_INVALID", "default"},
		{`{"type":"null","default":false}`, "E_SCHEMA_INVALID", "default"},
		{`{"type":"null","enum":[null],"default":{}}`, "E_SCHEMA_INVALID", "default"},
		{`{"enum":[null],"default":[]}`, "E_SCHEMA_INVALID", "default"},
		{`{"enum":[null,null],"default":null}`, "E_SCHEMA_INVALID", "enum/1"},
		{`{"type":"null","enum":[null,false],"default":null}`, "E_SCHEMA_INVALID", "enum/1"},
		{`{"type":"null","const":null,"default":null}`, "E_SCHEMA_UNSUPPORTED", "default"},
		{`{"const":null,"default":null}`, "E_SCHEMA_UNSUPPORTED", "default"},
		{`{"type":"null","enum":[null],"format":"","default":null}`, "E_SCHEMA_UNSUPPORTED", "format"},
		{`{"type":"null","x-native-type":"custom.Value","default":null}`, "E_SCHEMA_UNSUPPORTED", "default"},
		{`{"type":["null"],"default":null}`, "E_SCHEMA_UNSUPPORTED", "type"},
		{`{"$ref":"#/components/schemas/Value","default":null}`, "E_REF_MISSING", ""},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.value), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil {
				t.Fatal("Invalid default or widened grammar accepted", tc, validation)
			}
			d := err.(core.Diagnostics)[0]
			if d.Code != tc.code || d.Pointer == nil || *d.Pointer != strings.TrimSuffix("/components/schemas/Record/properties/value/"+tc.tail, "/") || !reflect.DeepEqual(got, core.GenerationReport{}) {
				if d.Pointer != nil {
					t.Log("Actual pointer:", *d.Pointer)
				}
				t.Fatal(tc, validation, err)
			}
		}
	}
}

func TestNullOnlyDefaultsDoNotAuthorizeCollectionLeaves(t *testing.T) {
	for _, input := range []string{
		`{"Holder":{"properties":{"values":{"type":"array","items":{"type":"null","default":null}}}}}`,
		`{"Holder":{"properties":{"values":{"additionalProperties":{"enum":[null],"default":null}}}}}`,
		`{"Holder":{"properties":{"value":{"type":"string"}},"additionalProperties":{"enum":[null],"default":null}}}`,
	} {
		r := request(schemas(input), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !strings.HasSuffix(*err.(core.Diagnostics)[0].Pointer, "/default") || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Field-only default permission leaked", input, err)
		}
	}
}
