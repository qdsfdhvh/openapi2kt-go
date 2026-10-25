package core_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTOEnumFieldUniqueActualConstructor(t *testing.T) {
	for _, field := range []string{`{"type":"string","enum":["ready","done"],"default":"ready"}`, `{"type":"integer","enum":[0,1],"default":0}`, `{"type":"boolean","enum":[false,true],"default":false}`} {
		input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"state":` + field + `},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !strings.Contains(source(t, r, "HolderDto"), "List<org.example.models.models.ItemDto>") {
			t.Fatal("actualDTO enumgetter uniqueness rejected", field, e)
		}
	}
}

func TestDTOEnumUniqueFormerScalarOnlyBoundary(t *testing.T) {
	input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"mode":{"type":"string","enum":["a","b"]}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	got, e := core.GenerateWithDiagnostics(rq)
	if e != nil || !strings.Contains(source(t, got, "HolderDto"), ".serialName") {
		t.Fatal("exact former enum-field source lost actual codec key", e)
	}
	rq.Options.Validation = false
	if got, e := core.GenerateWithDiagnostics(rq); e == nil || len(got.Artifacts.Files) != 0 {
		t.Fatal("enumDTO uniqueness requires optin", e)
	}
}

func TestDTOEnumUniqueConstructorDefaults(t *testing.T) {
	for _, tc := range []struct{ field, value string }{
		{`{"type":"string","enum":["ready","done"],"default":"ready"}`, `"ready"`},
		{`{"type":"integer","enum":[0,1],"default":0}`, `0`},
		{`{"type":"boolean","enum":[false,true],"default":false}`, `false`},
		{`{"type":"number","format":"float","enum":[0,1],"default":0}`, `-0`},
		{`{"type":["string","null"],"enum":["ready",null],"default":null}`, `null`},
	} {
		for _, required := range []bool{false, true} {
			suffix := ""
			if required {
				suffix = `,"required":["values"]`
			}
			input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"state":` + tc.field + `},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":[{"id":"a"},{"id":"a","state":` + tc.value + `}]}}` + suffix + `}}`)
			rq := request(input, "Dto")
			rq.Options.Validation = true
			r, e := core.GenerateWithDiagnostics(rq)
			ds, ok := e.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
				t.Fatal("constructor initializer duplicate lost original Pointer", tc.field, required, e)
			}
		}
	}
}

func TestDTOEnumUniqueOriginalVocabularyAndDefaults(t *testing.T) {
	for _, tc := range []struct{ field, pointer string }{
		{`{"type":"string","enum":["a","a"]}`, `/enum/1`},
		{`{"type":"string","enum":["a","b"],"default":"c"}`, `/default`},
		{`{"type":"integer","enum":[0,1],"default":2}`, `/default`},
		{`{"type":"integer","enum":[0,1],"minimum":2}`, `/enum`},
		{`{"type":"boolean","enum":[false,true],"default":0}`, `/default`},
	} {
		input := schemas(`{"Item":{"properties":{"state":` + tc.field + `},"required":["state"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/state"+tc.pointer || len(r.Artifacts.Files) != 0 {
			t.Fatal("strict enum annotation lost original Pointer", tc.field, e)
		}
	}
}

func TestDTOEnumUniqueVocabularyLimit(t *testing.T) {
	for _, kind := range []string{"string", "integer"} {
		for _, size := range []int{240, 241} {
			values := make([]any, size)
			for i := range size {
				if kind == "string" {
					values[i] = fmt.Sprint("v", i)
				} else {
					values[i] = i
				}
			}
			input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{
				"Item":   map[string]any{"properties": map[string]any{"state": map[string]any{"type": kind, "enum": values}}},
				"Holder": map[string]any{"properties": map[string]any{"values": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}}},
			}}})
			if err != nil {
				t.Fatal(err)
			}
			rq := request(input, "Dto")
			rq.Options.Validation = true
			r, e := core.GenerateWithDiagnostics(rq)
			if size == 240 {
				if e != nil {
					t.Fatal(kind, size, e)
				}
			} else {
				ds, ok := e.(core.Diagnostics)
				if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/state/enum" || len(r.Artifacts.Files) != 0 {
					t.Fatal("enum cap changed", kind, size, e)
				}
			}
		}
	}
}
