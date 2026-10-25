package core_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTOJsonNullGettersUnique(t *testing.T) {
	for _, field := range []string{`{"type":"null"}`, `{"type":"null","enum":[null]}`, `{"enum":[null],"default":null}`, `{"type":"null","const":null}`, `{"const":null}`} {
		input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"pin":` + field + `},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !strings.Contains(source(t, r, "HolderDto"), "List<org.example.models.models.ItemDto>") {
			if ds, ok := e.(core.Diagnostics); ok && len(ds) > 0 && ds[0].Pointer != nil {
				t.Fatal("actual JsonNull DTOgetter uniqueness rejected", field, e, *ds[0].Pointer)
			}
			t.Fatal("actual JsonNull DTOgetter uniqueness rejected", field, e)
		}
	}
}

func TestDTOJsonNullFormerConstBoundary(t *testing.T) {
	input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"pin":{"type":"null","const":null}}},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"}}}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal("exact former nullconst source", e)
	}
	rq.Options.Validation = false
	if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
		t.Fatal("JsonNullDTO unique optin lost", e)
	}
}

func TestDTOJsonNullUniqueVocabularyAndDefaultGuards(t *testing.T) {
	for _, tc := range []struct{ field, pointer string }{
		{`{"enum":[null,null]}`, `/enum/1`},
		{`{"enum":[null],"default":"null"}`, `/default`},
		{`{"type":"null","default":"null"}`, `/default`},
		{`{"const":null,"default":null}`, `/default`},
		{`{"type":"null","enum":[null,"null"]}`, `/enum/1`},
	} {
		input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"pin":` + tc.field + `}},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"}}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/pin"+tc.pointer || len(r.Artifacts.Files) != 0 {
			t.Fatal("JsonNull original grammar/default Pointer lost", tc.field, e)
		}
	}
}

func TestDTOJsonNullUniqueMemberLimit(t *testing.T) {
	for _, size := range []int{240, 241} {
		props := map[string]any{"id": map[string]any{"type": "string"}}
		for i := 1; i < size; i++ {
			props[fmt.Sprint("p", i)] = map[string]any{"type": "null"}
		}
		input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Item": map[string]any{"properties": props}, "Holder": map[string]any{"properties": map[string]any{"values": map[string]any{"type": "array", "uniqueItems": true, "items": map[string]any{"$ref": "#/components/schemas/Item"}}}}}}})
		if e != nil {
			t.Fatal(e)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		if size == 240 {
			if e != nil {
				t.Fatal("null getter exact member limit", e)
			}
		} else {
			ds, ok := e.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal("null getter member cap relaxed", e)
			}
		}
	}
}
