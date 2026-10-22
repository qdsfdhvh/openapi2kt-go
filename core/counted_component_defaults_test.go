package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestCountedNullableComponentDefaultFactories(t *testing.T) {
	for _, shape := range []string{
		`"properties":{"value":{"type":"string"}}`,
		`"allOf":[{"properties":{"value":{"type":"string"}},"minProperties":1}]`,
	} {
		for _, value := range []string{`null`, `{"value":"ready"}`} {
			input := schemas(`{"Record":{"type":["object","null"],` + shape + `,"minProperties":1,"maxProperties":1,"default":` + value + `},"Holder":{"properties":{"record":{"$ref":"#/components/schemas/Record"}}}}`)
			rq := request(input, "Dto")
			rq.Options.Validation = true
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(shape, value, err)
			}
			text := source(t, r, "RecordDto")
			if !strings.Contains(text, "public fun default(): RecordDto?") || !strings.Contains(text, "generatedSerializer()") {
				t.Fatal(text)
			}
			if strings.Contains(source(t, r, "HolderDto"), "RecordDto.default()") {
				t.Fatal("component annotation inherited by consumer")
			}
			if value != "null" {
				aliasFailure(t, []byte(strings.Replace(string(input), `"default":{"value":"ready"}`, `"default":{}`, 1)), "E_SCHEMA_INVALID", "/components/schemas/Record/default")
			}
		}
	}
}

func TestCountedComponentFactoriesDoNotGrantOtherSourceRoles(t *testing.T) {
	for _, body := range []string{
		`{"Target":{"nullable":true,"properties":{"value":{"type":"string"}},"minProperties":0},"Wrapper":{"$ref":"#/components/schemas/Target","default":null}}`,
		`{"Target":{"nullable":true,"properties":{"value":{"type":"string"}},"minProperties":0,"default":null},"Wrapper":{"$ref":"#/components/schemas/Target","default":null}}`,
		`{"Record":{"properties":{"value":{"nullable":true,"properties":{"name":{"type":"string"}},"minProperties":0,"default":null}}}}`,
		`{"Target":{"nullable":true,"allOf":[{"properties":{"value":{"type":"string"}},"minProperties":0}],"default":null},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Target","default":null}}}}`,
	} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = true
		if r, err := core.GenerateWithDiagnostics(rq); err == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("count default permission escaped original root", r, err)
		}
	}
	for _, bounds := range []string{`"minProperties":-1`, `"minProperties":2,"maxProperties":1`} {
		input := schemas(`{"Record":{"nullable":true,"allOf":[{"properties":{"value":{"type":"string"}},` + bounds + `}],"default":null}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		if _, err := core.Generate(rq); err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" {
			t.Fatal("null default hid invalid count source", err)
		}
	}
}
