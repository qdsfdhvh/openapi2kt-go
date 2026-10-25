package core_test

import (
	"encoding/json"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTONativeCollectionReferenceInlineControls(t *testing.T) {
	for _, leaf := range []string{`{"type":"integer","enum":[0,1]}`, `{"type":"number","format":"float","enum":[0,1]}`, `{"type":"boolean","enum":[true,false]}`} {
		rq := request(schemas(`{"Item":{"properties":{"value":{"type":"array","items":`+leaf+`}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("existing direct native collection key control", leaf, err)
		}
	}
}

func TestDTONativeCollectionReferenceUncheckedBorrowedGateUnchanged(t *testing.T) {
	for _, borrowed := range []bool{false, true} {
		definitions := `{"Value":{"type":"array","items":{"type":"integer","enum":[0,1]}}`
		reference := "Value"
		if borrowed {
			definitions = `{"Owner":{"properties":{"leaf":{"type":"array","items":{"type":"integer","enum":[0,1]}}}}`
			reference = "Owner/properties/leaf"
		}
		rq := request(schemas(definitions+`,"Item":{"properties":{"value":{"$ref":"#/components/schemas/`+reference+`"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"}}}}}`), "Dto")
		r, err := core.GenerateWithDiagnostics(rq)
		if !borrowed {
			if err != nil {
				t.Fatal("existing unchecked named collection refused", err)
			}
			continue
		}
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_REF_UNSUPPORTED" || len(r.Artifacts.Files) != 0 {
			t.Fatal("unchecked borrowed native gate changed", err)
		}
	}
}

func TestDTONativeCollectionReferenceVocabularyBudget(t *testing.T) {
	for _, count := range []int{240, 241} {
		vocabulary := make([]any, count)
		for i := range vocabulary {
			vocabulary[i] = i
		}
		definitions := map[string]any{"Value": map[string]any{"type": "array", "items": map[string]any{"type": "integer", "enum": vocabulary}}, "Item": map[string]any{"properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/Value", "default": []any{0}}}}, "Holder": map[string]any{"properties": map[string]any{"values": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}}}}
		data, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": definitions}})
		if e != nil {
			t.Fatal(e)
		}
		rq := request(data, "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if count == 240 {
			if err != nil {
				t.Fatal("raw240 vocabulary rejected", err)
			}
			continue
		}
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal("raw241 vocabulary accepted", err)
		}
	}
}

func TestDTONativeCollectionReferenceOwnDefaultsAndMappedKeys(t *testing.T) {
	for _, tc := range []struct{ leaf, value, explicit string }{
		{`{"type":"integer","enum":[0,1]}`, `[0]`, `[0]`},
		{`{"type":"number","format":"float","enum":[0,1]}`, `[0]`, `[-0.0]`},
		{`{"type":"boolean","enum":[true,false]}`, `[true]`, `[true]`},
	} {
		rq := request(schemas(`{"Value":{"type":"array","items":`+tc.leaf+`},"Alias":{"$ref":"#/components/schemas/Value"},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":`+tc.value+`}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":[{}, {"value":`+tc.explicit+`}]}}}}`), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			t.Fatal("native collection mapped initializer key", tc, err)
		}
	}
}

func TestDTONativeCollectionReferenceStrictDefaultBeforeRequired(t *testing.T) {
	for _, tc := range []struct{ leaf, value string }{
		{`{"type":"integer","enum":[0,1]}`, `[2]`},
		{`{"type":"number","format":"float","enum":[0.1]}`, `[0.1000000001]`},
		{`{"type":"boolean","enum":[true]}`, `[false]`},
	} {
		rq := request(schemas(`{"Value":{"type":"array","items":`+tc.leaf+`},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":`+tc.value+`}},"required":["value"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, "/components/schemas/Item/properties/value/default") || len(r.Artifacts.Files) != 0 {
			t.Fatal("native strict annotation before suppression", tc, err)
		}
	}
}

func TestDTONativeCollectionReferenceGetterUnique(t *testing.T) {
	for _, leaf := range []string{`{"type":"integer","enum":[0,1]}`, `{"type":"number","format":"float","enum":[0,1]}`, `{"type":"boolean","enum":[true,false]}`} {
		rq := request(schemas(`{"Value":{"type":"array","items":`+leaf+`},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("original native enum collection reference getter", leaf, err)
		}
	}
}
