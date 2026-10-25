package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTOScalarConstUniqueActualBody(t *testing.T) {
	for _, field := range []string{`{"const":"fixed"}`, `{"type":"integer","const":0}`, `{"type":"boolean","const":false}`, `{"type":"number","format":"float","const":-0}`} {
		input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"kind":` + field + `},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !strings.Contains(source(t, r, "HolderDto"), "List<org.example.models.models.ItemDto>") {
			t.Fatal("actual constant body DTO uniqueness rejected", field, e)
		}
	}
}

func TestDTOConstUniqueFormerBoundary(t *testing.T) {
	input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"pin":{"const":"a"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	r, e := core.GenerateWithDiagnostics(rq)
	if e != nil || !strings.Contains(source(t, r, "ItemDto"), `val pin: kotlin.String = "a"`) {
		t.Fatal("original const-body source did not retain body", e)
	}
	rq.Options.Validation = false
	if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
		t.Fatal("const DTO unique lost optin", e)
	}
}

func TestDTOConstUniqueOriginalDefaults(t *testing.T) {
	for _, values := range []string{`[{"id":"a"},{"id":"a","pin":"fixed"}]`, `[{"id":"a","pin":"other"}]`} {
		input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"pin":{"const":"fixed"}},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"},"default":` + values + `}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		pointer := "/components/schemas/Holder/properties/values/default/1"
		if strings.Contains(values, "other") {
			pointer = "/components/schemas/Holder/properties/values/default/0/pin"
		}
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != pointer || len(r.Artifacts.Files) != 0 {
			t.Fatal("const default validation/equality lost original Pointer", values, e)
		}
	}
}

func TestDTOConstUniqueKeepsExcludedBodies(t *testing.T) {
	for _, field := range []string{`{"const":{"a":1}}`, `{"const":[1,2]}`, `{"type":"string","x-native-type":"caller.Native","const":"fixed"}`} {
		input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"pin":` + field + `}},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"}}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("unproved constant codec authorized DTO equality", field, e)
		}
	}
}
