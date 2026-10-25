package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestDTOAllOfUniqueOriginalTypedControls(t *testing.T) {
	for _, item := range []string{`{"allOf":[{"properties":{"id":{"type":"string"}}}]}`, `{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"version":{"type":"integer","default":0}}}]}`} {
		for _, unique := range []bool{false, true} {
			check := ""
			if unique {
				check = `,"uniqueItems":true`
			}
			rq := request(schemas(`{"Base":{"properties":{"id":{"type":"string"}},"required":["id"]},"Item":`+item+`,"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"}`+check+`}}}}`), "Dto")
			rq.Options.Validation = true
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				t.Errorf("actual original allOf constructor unique=%v: %s %v", unique, item, e)
			}
		}
	}
}

func TestDTOAllOfUniqueOriginalFieldDefaultsAndNullable(t *testing.T) {
	for _, item := range []string{
		`{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"version":{"type":"integer","default":0}}}]}`,
		`{"allOf":[{"properties":{"id":{"type":"string"},"version":{"type":"integer","minimum":0,"default":0}}},{"properties":{"version":{"type":"integer","minimum":0,"default":1}}}],"required":["id"]}`,
		`{"nullable":true,"allOf":[{"$ref":"#/components/schemas/Base"},{"nullable":true,"properties":{"version":{"type":"integer","default":0}}}]}`,
	} {
		for _, duplicate := range []bool{false, true} {
			value := `[{"id":"a"},{"id":"b"}]`
			if duplicate {
				value = `[{"id":"a"},{"id":"a"}]`
			}
			rq := request(schemas(`{"Base":{"nullable":true,"properties":{"id":{"type":"string"}},"required":["id"]},"Item":`+item+`,"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":`+value+`}}}}`), "Dto")
			rq.Options.Validation = true
			r, e := core.GenerateWithDiagnostics(rq)
			if !duplicate {
				if e != nil {
					t.Fatal("original allOf field default", item, e)
				}
				continue
			}
			ds, ok := e.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
				t.Fatal("actual allOf constructor default key", item, e)
			}
		}
	}
}

func TestDTOAllOfUniqueOriginalReferenceCollectionGetters(t *testing.T) {
	rq := request(schemas(`{"Scalar":{"type":"string","enum":["ready","done"]},"Base":{"properties":{"states":{"type":"array","items":{"$ref":"#/components/schemas/Scalar"},"default":["ready"]}}},"Item":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"id":{"type":"string"}},"required":["id"]}]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
	rq.Options.Validation = true
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal("actual inherited collection reference getter", e)
	}
}

func dtoAllOfUniqueGraduation(t *testing.T, input []byte, holder string) {
	t.Helper()
	rq := request(input, "Dto")
	rq.Options.Validation = true
	r, e := core.GenerateWithDiagnostics(rq)
	if e != nil {
		t.Fatal("exact former allOf DTO refusal", e)
	}
	text := source(t, r, holder)
	if !strings.Contains(text, "kotlin.collections.List<") || strings.Contains(text, "`openapi-kt unique JSON items`") {
		t.Fatal("actual allOf DTO lost typed getter equality", text)
	}
	rq.Options.Validation = false
	if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
		t.Fatal("lost explicit validation gate", e)
	}
}

func TestDTOAllOfUniqueOriginalSourceAndCheckedBudgets(t *testing.T) {
	for _, tc := range []struct {
		aliases, depth int
		accepted       bool
	}{{0, 15, true}, {0, 16, false}, {29, 15, true}, {30, 15, false}} {
		defs := map[string]any{"Scalar": map[string]any{"type": "string", "enum": []any{"ready", "done"}}}
		last := "Scalar"
		for i := 0; i < tc.aliases; i++ {
			name := fmt.Sprint("A", i)
			defs[name] = map[string]any{"$ref": "#/components/schemas/" + last}
			last = name
		}
		var leaf any = map[string]any{"$ref": "#/components/schemas/" + last}
		for i := 0; i < tc.depth; i++ {
			leaf = map[string]any{"type": "array", "items": leaf}
		}
		defs["Value"] = leaf
		defs["Base"] = map[string]any{"properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/Value"}}}
		defs["Item"] = map[string]any{"allOf": []any{map[string]any{"$ref": "#/components/schemas/Base"}, map[string]any{"properties": map[string]any{}}}}
		defs["Holder"] = map[string]any{"properties": map[string]any{"values": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}}}
		data, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": defs}})
		if e != nil {
			t.Fatal(e)
		}
		rq := request(data, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		if tc.accepted && e != nil || !tc.accepted && (e == nil || len(r.Artifacts.Files) != 0) {
			t.Fatal("allOf source48/checked16 with original hops", tc, e)
		}
	}
}

func TestDTOAllOfUniqueOriginalNestedOwners(t *testing.T) {
	for _, owner := range []string{
		`{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]},{"properties":{"version":{"type":"integer","default":0}}}]}}}`,
		`{"allOf":[{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]},{"properties":{"version":{"type":"integer","default":0}}}]}}},{"properties":{"marker":{"type":"string"}}}]}`,
	} {
		reference := "#/components/schemas/Owner/properties/payload"
		if strings.HasPrefix(owner, `{"allOf"`) {
			reference = "#/components/schemas/Owner/allOf/0/properties/payload"
		}
		rq := request(schemas(`{"Owner":`+owner+`,"Bridge":{"$ref":"`+reference+`"},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Bridge"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("actual nested allOf constructor", owner, e)
		}
	}
}
