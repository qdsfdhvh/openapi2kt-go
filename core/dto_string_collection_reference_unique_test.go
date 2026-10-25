package core_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTOStringCollectionReferenceInlineControl(t *testing.T) {
	rq := request(schemas(`{"Item":{"properties":{"value":{"type":"array","items":{"type":"string","enum":["a","b"]}}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("existing direct String enum collection control", err)
	}
}

func TestDTOStringCollectionReferenceInternalBridgeAndOwnDefaults(t *testing.T) {
	definitions := `{"Value":{"type":"array","items":{"type":"string","enum":["a","b"]}},"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Owner":{"properties":{"a/b":{"type":"object","additionalProperties":{"type":"string","enum":["a","b"]}}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/a~1b"}`
	for _, field := range []string{`{"$ref":"#/components/schemas/Alias","nullable":false,"default":["a"]}`, `{"$ref":"#/components/schemas/Owner/properties/a~1b","default":{"x":"a"}}`, `{"$ref":"#/components/schemas/Bridge","default":{"x":"a"}}`} {
		for _, required := range []string{`[]`, `["value"]`} {
			rq := request(schemas(definitions+`,"Item":{"properties":{"value":`+field+`},"required":`+required+`},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
			rq.Options.Validation = true
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("actual enum collection reference/default source", field, required, err)
			}
		}
	}
}

func TestDTOStringCollectionReferenceOwnDefaultKeysAndStrictAnnotations(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		annotation := `["a"]`
		if invalid {
			annotation = `["future"]`
		}
		rq := request(schemas(`{"Value":{"type":"array","items":{"type":"string","enum":["a","b"]}},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":`+annotation+`}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":[{}, {"value":["a"]}]}}}}`), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		ds, ok := err.(core.Diagnostics)
		pointer := "/components/schemas/Holder/properties/values/default/1"
		if invalid {
			pointer = "/components/schemas/Item/properties/value/default/0"
		}
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != pointer || len(r.Artifacts.Files) != 0 {
			t.Fatal("strict original String vocabulary vs mapped default key", invalid, err)
		}
	}
	for _, required := range []bool{false, true} {
		definition := `{"Value":{"type":"array","items":{"type":"string","enum":["a","b"]}},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":["future"]}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`
		if required {
			definition = strings.Replace(definition, `"default":["future"]}}}`, `"default":["future"]}},"required":["value"]}`, 1)
		}
		rq := request(schemas(definition), "Dto")
		rq.Options.Validation = true
		if r, err := core.GenerateWithDiagnostics(rq); err == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("required suppressed invalid original vocabulary", required, err)
		}
	}
}

func TestDTOStringCollectionReferenceGetterUnique(t *testing.T) {
	for _, field := range []string{`{"$ref":"#/components/schemas/Value"}`, `{"$ref":"#/components/schemas/Value","default":[]}`} {
		rq := request(schemas(`{"Value":{"type":"array","items":{"type":"string","enum":["a","b"]}},"Item":{"properties":{"value":`+field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("exact original String enum collection reference getter", field, err)
		}
	}
}

func TestDTOStringCollectionReferenceDepthBoundaries(t *testing.T) {
	for _, tc := range []struct {
		aliases, depth int
		accepted       bool
	}{{0, 15, true}, {0, 16, false}, {31, 15, true}, {32, 15, false}} {
		var leaf any = map[string]any{"type": "string", "enum": []any{"ready", "done"}}
		for range tc.depth {
			leaf = map[string]any{"type": "array", "items": leaf}
		}
		definitions := map[string]any{"Value": leaf}
		last := "Value"
		for i := 0; i < tc.aliases; i++ {
			name := fmt.Sprint("A", i)
			definitions[name] = map[string]any{"$ref": "#/components/schemas/" + last}
			last = name
		}
		definitions["Item"] = map[string]any{"properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/" + last}}}
		definitions["Holder"] = map[string]any{"properties": map[string]any{"values": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}}}
		data, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": definitions}})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(data, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		if tc.accepted && e != nil || !tc.accepted && (e == nil || len(r.Artifacts.Files) != 0) {
			t.Fatal("reference collection source/checked boundary", tc, e)
		}
	}
}

func TestDTOStringCollectionReferenceVocabularyBudget(t *testing.T) {
	for _, count := range []int{240, 241} {
		vocabulary := make([]any, count)
		for i := range vocabulary {
			vocabulary[i] = fmt.Sprint("member", i)
		}
		definitions := map[string]any{"Value": map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": vocabulary}}, "Item": map[string]any{"properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/Value", "default": []any{"member0"}}}}, "Holder": map[string]any{"properties": map[string]any{"values": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}}}}
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
		if !ok || len(ds) != 1 || ds[0].Code != "E_ENUM_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal("raw241 vocabulary accepted", err)
		}
	}
}
