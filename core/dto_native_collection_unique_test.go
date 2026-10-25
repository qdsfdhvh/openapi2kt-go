package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestDTONativeEnumCollectionGettersUnique(t *testing.T) {
	for _, leaf := range []string{`{"type":"integer","format":"int32","enum":[0,1]}`, `{"type":"number","format":"float","enum":[0,1]}`, `{"type":"boolean","enum":[false,true]}`} {
		rq := request(schemas(`{"Item":{"properties":{"values":{"type":"array","items":`+leaf+`}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("actual native enum collection getter", leaf, e)
		}
	}
}

func TestDTONativeEnumCollectionMappedDefaultKeys(t *testing.T) {
	for _, tc := range []struct{ field, values string }{
		{`{"type":"array","items":{"type":"number","format":"float","enum":[16777216,16777217]}}`, `[{"values":[16777216]},{"values":[16777217]}]`},
		{`{"type":"object","additionalProperties":{"type":"boolean","enum":[false,true]},"default":{"a":false,"b":true}}`, `[{}, {"values":{"b":true,"a":false}}]`},
		{`{"type":"array","items":{"type":"integer","format":"int64","enum":[0,1]},"default":[0,1]}`, `[{}, {"values":[0,1]}]`},
	} {
		rq := request(schemas(`{"Item":{"properties":{"values":`+tc.field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":`+tc.values+`}}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			t.Fatal("mapped nativeenum collection key", tc.field, e)
		}
	}
}
func TestDTONativeEnumCollectionOriginalFailures(t *testing.T) {
	for _, tc := range []struct{ field, tail string }{
		{`{"type":"array","items":{"type":"integer","enum":[0,0]}}`, "/items/enum/1"},
		{`{"type":"array","items":{"type":"number","format":"float","enum":[0,1]},"default":[2]}`, "/default/0"},
		{`{"type":"array","items":{"type":"integer","enum":[0,1],"minimum":"bad"}}`, "/items/minimum"},
		{`{"type":"array","items":{"type":"integer","enum":[0,1],"default":0}}`, "/items/default"},
	} {
		rq := request(schemas(`{"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}},"Item":{"properties":{"values":`+tc.field+`}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/values"+tc.tail || len(r.Artifacts.Files) != 0 {
			if ok && len(ds) > 0 && ds[0].Pointer != nil {
				t.Fatal("nativeenum source failure masked", tc.field, e, *ds[0].Pointer)
			}
			t.Fatal("nativeenum source failure", tc.field, e)
		}
	}
}
func TestDTONativeEnumCollectionSharesCheckedDepth(t *testing.T) {
	for _, outer := range []int{1, 2, 8} {
		for _, total := range []int{16, 17} {
			field := map[string]any{"type": "integer", "enum": []any{0, 1}}
			for range total - outer {
				field = map[string]any{"type": "array", "items": field}
			}
			values := map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}
			for range outer - 1 {
				values = map[string]any{"type": "array", "items": values}
			}
			input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Item": map[string]any{"properties": map[string]any{"id": map[string]any{"type": "integer", "enum": []any{0, 1}}, "values": field}}, "Holder": map[string]any{"properties": map[string]any{"values": values}}}}})
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
