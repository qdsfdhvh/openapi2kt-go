package core_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTOBuiltinCollectionReferenceInlineControl(t *testing.T) {
	rq := request(schemas(`{"Item":{"properties":{"value":{"type":"array","items":{"type":"integer"}}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("existing inline getter control", err)
	}
}

func TestDTOBuiltinCollectionReferenceDepthBoundaries(t *testing.T) {
	for _, tc := range []struct {
		aliases, depth int
		accepted       bool
	}{{0, 15, true}, {0, 16, false}, {31, 15, true}, {32, 15, false}} {
		var leaf any = map[string]any{"type": "integer"}
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

func TestDTOBuiltinCollectionReferenceUnadmittedSourcesRemainClosed(t *testing.T) {
	for _, definition := range []string{
		`{"Value":{"type":"array","items":{"type":"integer"},"default":[0]}}`,
		`{"Value":{"type":"array","items":{"properties":{"id":{"type":"string"}}}}}`,
		`{"Value":{"$ref":"#/components/schemas/Alias"},"Alias":{"type":"array","items":{"type":"integer"},"default":[0]}}`,
	} {
		for _, field := range []string{`{"$ref":"#/components/schemas/Value"}`, `{"$ref":"#/components/schemas/Value","default":[]}`} {
			rq := request(schemas(definition[:len(definition)-1]+`,"Item":{"properties":{"value":`+field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
			rq.Options.Validation = true
			if r, err := core.GenerateWithDiagnostics(rq); err == nil || len(r.Artifacts.Files) != 0 {
				t.Fatal("unadmitted referenced tree accepted", definition, field, err)
			}
		}
	}
}

func TestDTOBuiltinCollectionReferenceGetterUnique(t *testing.T) {
	for _, defs := range []string{
		`{"Value":{"type":"array","items":{"type":"integer"}}}`,
		`{"Value":{"type":"object","additionalProperties":{"type":"number","format":"float"}}}`,
		`{"Value":{"$ref":"#/components/schemas/Hops/properties/leaf"},"Hops":{"properties":{"leaf":{"type":"array","items":{"type":"string"}}}}}`,
	} {
		rq := request(schemas(defs[:len(defs)-1]+`,"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("original referenced builtin collection getter", defs, err)
		}
	}
}

func TestDTOBuiltinCollectionReferenceOwnDefaultMappedKeys(t *testing.T) {
	for _, tc := range []struct{ definition, field, values string }{
		{`{"Value":{"type":"array","items":{"type":"number","format":"float"}}}`, `{"$ref":"#/components/schemas/Value","default":[0,1]}`, `[{}, {"value":[-0.0,1]}]`},
		{`{"Value":{"type":"object","additionalProperties":{"type":"number","format":"float"}},"Alias":{"$ref":"#/components/schemas/Value"}}`, `{"$ref":"#/components/schemas/Alias","default":{"a":0,"b":1}}`, `[{}, {"value":{"b":1,"a":-0.0}}]`},
		{`{"Value":{"type":"array","items":{"type":"object","additionalProperties":{"type":"array","items":{"type":"number","format":"double","nullable":true}}}}}`, `{"$ref":"#/components/schemas/Value","default":[{"a":[0,null]}]}`, `[{}, {"value":[{"a":[-0.0,null]}]}]`},
	} {
		rq := request(schemas(tc.definition[:len(tc.definition)-1]+`,"Item":{"properties":{"value":`+tc.field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":`+tc.values+`}}}}`), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			t.Fatal("mapped referenced initializer equality", tc.field, err)
		}
	}
}

func TestDTOBuiltinCollectionReferenceNullableRequiredAndDefaults(t *testing.T) {
	for _, required := range []string{`[]`, `["value"]`} {
		for _, field := range []string{`{"$ref":"#/components/schemas/Alias","default":null}`, `{"$ref":"#/components/schemas/Alias","nullable":false,"default":[1]}`} {
			rq := request(schemas(`{"Value":{"type":"array","nullable":true,"items":{"type":"integer"}},"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Item":{"properties":{"value":`+field+`},"required":`+required+`},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
			rq.Options.Validation = true
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal(field, required, err)
			}
		}
	}
}

func TestDTOBuiltinCollectionReferenceDefaultValidationBeforeRequired(t *testing.T) {
	for _, field := range []string{`{"$ref":"#/components/schemas/Value","default":[0]}`, `{"$ref":"#/components/schemas/Value","nullable":false,"default":null}`} {
		rq := request(schemas(`{"Value":{"type":"array","items":{"type":"integer","minimum":1}},"Item":{"properties":{"value":`+field+`},"required":["value"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, "/components/schemas/Item/properties/value/default") || len(r.Artifacts.Files) != 0 {
			t.Fatal("required default annotation was not validated", field, err)
		}
	}
}
