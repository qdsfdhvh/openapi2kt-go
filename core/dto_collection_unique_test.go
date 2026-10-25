package core_test

import (
	"encoding/json"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTOInlineScalarCollectionUnique(t *testing.T) {
	for _, field := range []string{`{"type":"array","items":{"type":"number","format":"float"}}`, `{"type":"object","additionalProperties":{"type":"string"}}`, `{"type":"array","items":{"type":"integer"},"default":[0,1]}`} {
		input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"values":` + field + `},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"}}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !strings.Contains(source(t, r, "HolderDto"), "List<org.example.models.models.ItemDto>") {
			t.Fatal("actualDTO scalar collection getter uniqueness rejected", field, e)
		}
	}
}

func TestDTOCollectionUniqueMappedDefaults(t *testing.T) {
	for _, tc := range []struct{ field, values string }{
		{`{"type":"array","items":{"type":"number","format":"float"},"default":[0]}`, `[{"id":"a"},{"id":"a","values":[-0.0]}]`},
		{`{"type":"object","additionalProperties":{"type":"number","format":"float"}}`, `[{"id":"a","values":{"a":0,"b":1}},{"id":"a","values":{"b":1,"a":-0}}]`},
		{`{"type":"array","items":{"type":"object","additionalProperties":{"type":"string","format":"uuid"}}}`, `[{"id":"a","values":[{"a":"550E8400-E29B-41D4-A716-446655440000"}]},{"id":"a","values":[{"a":"550e8400-e29b-41d4-a716-446655440000"}]}]`},
	} {
		input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"values":` + tc.field + `},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"},"default":` + tc.values + `}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			if ok && len(ds) > 0 && ds[0].Pointer != nil {
				t.Fatal("mapped collection default Pointer", tc.field, e, *ds[0].Pointer)
			}
			t.Fatal("mapped collection default equality", tc.field, e)
		}
	}
}

func TestDTOCollectionUniqueSharesCheckedDepth(t *testing.T) {
	for _, outer := range []int{1, 2, 8} {
		for _, total := range []int{16, 17} {
			field := map[string]any{"type": "string"}
			for range total - outer {
				field = map[string]any{"type": "array", "items": field}
			}
			values := map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}
			for range outer - 1 {
				values = map[string]any{"type": "array", "items": values}
			}
			input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Item": map[string]any{"properties": map[string]any{"id": map[string]any{"type": "string"}, "values": field}}, "Holder": map[string]any{"properties": map[string]any{"values": values}}}}})
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

func TestDTOCollectionUniqueLeavesAndDataRemainSeparate(t *testing.T) {
	for _, field := range []string{
		`{"type":"array","items":{"properties":{"id":{"type":"string"}}}}`,
		`{"type":"object","additionalProperties":true}`,
		`{"type":"array","items":{"oneOf":[{"type":"integer"},{"type":"null"}]}}`,
	} {
		input := schemas(`{"Number":{"type":"integer"},"Item":{"properties":{"id":{"type":"string"},"values":` + field + `}},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"}}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("unproved container leaf/ref/raw role inferred", field, e)
		}
	}
	input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"values":{"type":"object","additionalProperties":{"type":"number","format":"float"},"default":{"$ref":0,"type":1}}}},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"}}}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal("map default data keys were interpreted as schema references", e)
	}
}

func TestDTOCollectionUniqueOriginalSourceFailures(t *testing.T) {
	for _, tc := range []struct{ field, pointer string }{
		{`{"type":"array","items":{"type":"number","minimum":"bad"}}`, `/items/minimum`},
		{`{"type":"array","items":{"type":"integer"},"default":["bad"]}`, `/default/0`},
		{`{"type":"array","items":{"type":"integer","default":0}}`, `/items/default`},
	} {
		input := schemas(`{"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"}}}},"Item":{"properties":{"id":{"type":"string"},"values":` + tc.field + `}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/values"+tc.pointer || len(r.Artifacts.Files) != 0 {
			if ok && len(ds) > 0 && ds[0].Pointer != nil {
				t.Fatal("original source failure hidden", tc.field, e, *ds[0].Pointer)
			}
			t.Fatal("original source failure hidden", tc.field, e)
		}
	}
}
