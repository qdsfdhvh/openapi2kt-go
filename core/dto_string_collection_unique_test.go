package core_test

import (
	"encoding/json"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTOStringEnumCollectionGettersUnique(t *testing.T) {
	for _, field := range []string{`{"type":"array","items":{"enum":["a","b"]}}`, `{"type":"object","additionalProperties":{"type":"array","items":{"type":"string","enum":["a","b"],"nullable":true}},"default":{"one":["a",null]}}`} {
		rq := request(schemas(`{"Item":{"properties":{"values":`+field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("actual String enum collection getter", field, e)
		}
	}
}

func TestDTOStringEnumCollectionFormerBoundary(t *testing.T) {
	input := schemas(`{"Number":{"type":"integer"},"Item":{"properties":{"id":{"type":"string"},"values":{"type":"array","items":{"enum":["a","b"]}}}},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"}}}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal("exact former String enum collection getter boundary", e)
	}
}

func TestDTOStringEnumCollectionDefaultKeys(t *testing.T) {
	for _, tc := range []struct{ field, values string }{
		{`{"type":"array","items":{"enum":["a","b"]},"default":["a"]}`, `[{}, {"values":["a"]}]`},
		{`{"type":"object","additionalProperties":{"type":"array","items":{"enum":["a","b"],"nullable":true}},"default":{"a":["a",null],"b":[]}}`, `[{}, {"values":{"b":[],"a":["a",null]}}]`},
	} {
		rq := request(schemas(`{"Item":{"properties":{"values":`+tc.field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":`+tc.values+`}}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			t.Fatal("String enum collection initializer key", tc.field, e)
		}
	}
}
func TestDTOStringEnumCollectionOriginalFailures(t *testing.T) {
	for _, tc := range []struct{ field, tail string }{
		{`{"type":"array","items":{"enum":["a","a"]}}`, "/items/enum/1"},
		{`{"type":"array","items":{"enum":["a","b"]},"default":["future"]}`, "/default/0"},
		{`{"type":"array","items":{"enum":["a","b"],"default":"a"}}`, "/items/default"},
	} {
		rq := request(schemas(`{"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}},"Item":{"properties":{"values":`+tc.field+`}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/values"+tc.tail || len(r.Artifacts.Files) != 0 {
			t.Fatal("String enum source failure", tc.field, e)
		}
	}
}
func TestDTOStringEnumCollectionSharesCheckedDepth(t *testing.T) {
	for _, outer := range []int{1, 2, 8} {
		for _, total := range []int{16, 17} {
			field := map[string]any{"type": "string", "enum": []any{"a", "b"}}
			for range total - outer {
				field = map[string]any{"type": "array", "items": field}
			}
			values := map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}
			for range outer - 1 {
				values = map[string]any{"type": "array", "items": values}
			}
			input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Item": map[string]any{"properties": map[string]any{"id": map[string]any{"type": "string", "enum": []any{"a", "b"}}, "values": field}}, "Holder": map[string]any{"properties": map[string]any{"values": values}}}}})
			if e != nil {
				t.Fatal(e)
			}
			rq := request(input, "Dto")
			rq.Options.Validation = true
			r, e := core.GenerateWithDiagnostics(rq)
			if total == 16 {
				if e != nil {
					t.Fatal("exact getter checkeddepth16", outer, e)
				}
			} else {
				ds, ok := e.(core.Diagnostics)
				pointer := "/components/schemas/Holder/properties/values" + strings.Repeat("/items", outer-1)
				if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_DEPTH" || ds[0].Pointer == nil || *ds[0].Pointer != pointer || len(r.Artifacts.Files) != 0 {
					if ok && len(ds) > 0 && ds[0].Pointer != nil {
						t.Fatal("getter tree reset checkeddepth", outer, e, *ds[0].Pointer)
					}
					t.Fatal("getter tree reset checkeddepth", outer, e)
				}
			}
		}
	}
}

func TestDTOStringEnumCollectionNullTokenStates(t *testing.T) {
	rq := request(schemas(`{"Item":{"properties":{"values":{"type":"array","items":{"enum":["null","a"],"nullable":true}}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":[{"values":[null]},{"values":["null"]}]}}}}`), "Dto")
	rq.Options.Validation = true
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal("JSON null merged with String null getter state", e)
	}
}
