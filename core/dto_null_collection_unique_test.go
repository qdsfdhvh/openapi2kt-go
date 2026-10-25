package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestDTOJsonNullCollectionGettersUnique(t *testing.T) {
	for _, leaf := range []string{`{"type":"null"}`, `{"enum":[null]}`, `{"type":"null","enum":[null],"nullable":true}`} {
		input := schemas(`{"Item":{"properties":{"values":{"type":"array","items":` + leaf + `}},"required":["values"]},"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("actual JsonNull collection getter", leaf, e)
		}
	}
}

func TestDTOJsonNullCollectionFormerBoundary(t *testing.T) {
	input := schemas(`{"Number":{"type":"integer"},"Item":{"properties":{"id":{"type":"string"},"values":{"type":"array","items":{"type":"null"}}}},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"}}}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal("exact former JsonNull collection getter boundary", e)
	}
}

func TestDTOJsonNullCollectionDefaultKeys(t *testing.T) {
	for _, tc := range []struct{ field, values string }{
		{`{"type":"array","items":{"type":"null"},"default":[null]}`, `[{"id":"a"},{"id":"a","values":[null]}]`},
		{`{"type":"array","items":{"enum":[null],"nullable":true},"default":[null]}`, `[{"id":"a"},{"id":"a","values":[null]}]`},
		{`{"type":"object","additionalProperties":{"type":"array","items":{"type":"null"}}}`, `[{"id":"a","values":{"a":[null],"b":[]}},{"id":"a","values":{"b":[],"a":[null]}}]`},
	} {
		rq := request(schemas(`{"Item":{"properties":{"id":{"type":"string"},"values":`+tc.field+`},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":`+tc.values+`}}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			t.Fatal("null collection initializer/default equality", tc.field, e)
		}
	}
}
func TestDTONullCollectionUniqueSharesCheckedDepth(t *testing.T) {
	for _, outer := range []int{1, 2, 8} {
		for _, total := range []int{16, 17} {
			field := map[string]any{"type": "null"}
			for range total - outer {
				field = map[string]any{"type": "array", "items": field}
			}
			values := map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}
			for range outer - 1 {
				values = map[string]any{"type": "array", "items": values}
			}
			input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Item": map[string]any{"properties": map[string]any{"id": map[string]any{"type": "null"}, "values": field}}, "Holder": map[string]any{"properties": map[string]any{"values": values}}}}})
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

func TestDTOJsonNullCollectionOriginalFailurePointers(t *testing.T) {
	for _, tc := range []struct{ field, tail string }{
		{`{"type":"array","items":{"enum":[null,null]}}`, "/items/enum/1"},
		{`{"type":"array","items":{"type":"null"},"default":["null"]}`, "/default/0"},
		{`{"type":"array","items":{"type":"null","default":null}}`, "/items/default"},
	} {
		rq := request(schemas(`{"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}},"Item":{"properties":{"values":`+tc.field+`}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/values"+tc.tail || len(r.Artifacts.Files) != 0 {
			if ok && len(ds) > 0 && ds[0].Pointer != nil {
				t.Fatal("source failure masked", tc.field, e, *ds[0].Pointer)
			}
			t.Fatal("source failure", tc.field, e)
		}
	}
}

func TestDTOJsonNullCollectionRawEnumLimit(t *testing.T) {
	for _, size := range []int{240, 241} {
		vocab := make([]any, size)
		doc := map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Item": map[string]any{"properties": map[string]any{"values": map[string]any{"type": "array", "items": map[string]any{"enum": vocab}, "default": []any{}}}}, "Holder": map[string]any{"properties": map[string]any{"values": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}}}}}}
		data, e := json.Marshal(doc)
		if e != nil {
			t.Fatal(e)
		}
		rq := request(data, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		code, pointer := "E_SCHEMA_INVALID", "/components/schemas/Item/properties/values/items/enum/1"
		if size == 241 {
			code = "E_ENUM_LIMIT"
			pointer = "/components/schemas/Item/properties/values/items/enum"
		}
		if !ok || len(ds) != 1 || ds[0].Code != code || ds[0].Pointer == nil || *ds[0].Pointer != pointer || len(r.Artifacts.Files) != 0 {
			t.Fatal("null getter rawenum cap/precedence", size, e)
		}
	}
}
