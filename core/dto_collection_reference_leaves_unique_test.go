package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"testing"
)

func TestDTOCollectionReferenceLeavesTypedControlsAndUnique(t *testing.T) {
	for _, scalar := range []string{`{"type":"integer"}`, `{"type":"integer","enum":[0,1]}`, `{"type":"string","enum":["ready","done"]}`, `{"type":"string","format":"uuid"}`} {
		for _, rootref := range []bool{false, true} {
			for _, unique := range []bool{false, true} {
				collection := `{"type":"object","additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/Alias","nullable":false}}}`
				field := collection
				if rootref {
					field = `{"$ref":"#/components/schemas/Value"}`
				}
				check := ""
				if unique {
					check = `,"uniqueItems":true`
				}
				rq := request(schemas(`{"Scalar":`+scalar+`,"Alias":{"$ref":"#/components/schemas/Scalar","nullable":true},"Value":`+collection+`,"Item":{"properties":{"value":`+field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"}`+check+`}}}}`), "Dto")
				rq.Options.Validation = true
				if _, err := core.GenerateWithDiagnostics(rq); err != nil {
					t.Errorf("original leaf-reference typed/unique %s rootref=%v unique=%v: %v", scalar, rootref, unique, err)
				}
			}
		}
	}
}

func TestDTOCollectionReferenceLeavesExactFormerRefusal(t *testing.T) {
	for _, field := range []string{`{"$ref":"#/components/schemas/Value"}`, `{"$ref":"#/components/schemas/Value","default":[]}`} {
		rq := request(schemas(`{"Value":{"type":"array","items":{"$ref":"#/components/schemas/Scalar"}},"Scalar":{"type":"integer"},"Item":{"properties":{"value":`+field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("exact former collection-leaf-reference refusal", field, e)
		}
	}
}

func TestDTOCollectionReferenceLeavesOwnDefaults(t *testing.T) {
	for _, field := range []string{`{"type":"array","items":{"$ref":"#/components/schemas/Scalar"},"default":["ready"]}`, `{"$ref":"#/components/schemas/Value","default":["ready"]}`} {
		rq := request(schemas(`{"Scalar":{"type":"string","enum":["ready","done"]},"Value":{"type":"array","items":{"$ref":"#/components/schemas/Scalar"}},"Item":{"properties":{"value":`+field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Error("original reference-leaf own default", field, e)
		}
	}
}

func TestDTOCollectionReferenceLeavesInternalBridgeAndStrictDefaults(t *testing.T) {
	definitions := `{"Owner":{"properties":{"e/num":{"type":"string","enum":["ready","done"]},"ints":{"type":"integer","minimum":0}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/e~1num"},"Value":{"type":"array","items":{"$ref":"#/components/schemas/Bridge"}}`
	for _, field := range []string{`{"$ref":"#/components/schemas/Value","default":["ready"]}`, `{"type":"object","additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/e~1num"}},"default":{"a":["ready"]}}`} {
		for _, required := range []string{`[]`, `["value"]`} {
			rq := request(schemas(definitions+`,"Item":{"properties":{"value":`+field+`},"required":`+required+`},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
			rq.Options.Validation = true
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				if ds, ok := e.(core.Diagnostics); ok {
					for _, d := range ds {
						if d.Pointer != nil {
							t.Log("Pointer", *d.Pointer)
						}
					}
				}
				t.Fatal(field, required, e)
			}
		}
	}
	for _, required := range []string{`[]`, `["value"]`} {
		rq := request(schemas(definitions+`,"Item":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Bridge"},"default":["future"]}},"required":`+required+`},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/value/default/0" || len(r.Artifacts.Files) != 0 {
			t.Fatal("strict original default before suppression", required, e)
		}
	}
}

func TestDTOCollectionReferenceLeavesMappedDefaultKey(t *testing.T) {
	for _, field := range []string{`{"type":"array","items":{"$ref":"#/components/schemas/Scalar"},"default":[0]}`, `{"$ref":"#/components/schemas/Value","default":[0]}`} {
		rq := request(schemas(`{"Scalar":{"type":"number","format":"float"},"Value":{"type":"array","items":{"$ref":"#/components/schemas/Scalar"}},"Item":{"properties":{"value":`+field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":[{}, {"value":[-0.0]}]}}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			if ok {
				for _, d := range ds {
					if d.Pointer != nil {
						t.Log("Pointer", *d.Pointer)
					}
				}
			}
			t.Fatal("mapped constructor initializer key", field, e)
		}
	}
}

func TestDTOCollectionReferenceLeavesExactOldDefaultGraduations(t *testing.T) {
	for _, tc := range []struct {
		input  string
		native bool
	}{
		{`{"Code":{"type":"integer","enum":[1,2]},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"default":[1]}}}}`, true},
		{`{"Leaf":{"type":"string"},"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}},"default":[]}}}}`, false},
		{`{"Leaf":{"enum":["one"]},"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}},"default":[]}}}}`, false},
		{`{"Day":{"type":"string","format":"date"},"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"$ref":"#/components/schemas/Day"}},"default":[]}}}}`, false},
		{`{"Code":{"type":"string","enum":["one"]},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"default":["one"]}}}}`, false},
		{`{"Day":{"type":"string","format":"date"},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Day"},"default":[]}}}}`, false},
		{`{"Day":{"type":"string","format":"date"},"Holder":{"properties":{"value":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Day"},"default":{}}}}}`, false},
		{`{"Code":{"type":"integer","enum":[1,2]},"Holder":{"properties":{"value":` + nestedDefaultChain(2, `{"$ref":"#/components/schemas/Code"}`, `[]`) + `}}}`, true},
		{`{"Catalog":{"properties":{"code":{"type":"integer","enum":[1,2]}}},"Holder":{"properties":{"value":` + nestedDefaultChain(2, `{"$ref":"#/components/schemas/Catalog/properties/code"}`, `[]`) + `}}}`, true},
		{`{"Day":{"type":"string","format":"date"},"Holder":{"properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Day"},"default":[]}}}}`, false},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(tc.input), "Dto")
			rq.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(rq)
			if tc.native && !checked {
				if e == nil || len(r.Artifacts.Files) != 0 {
					t.Fatal("original unchecked native default gate", tc.input, e)
				}
				continue
			}
			if e != nil || len(r.Artifacts.Files) == 0 {
				t.Fatal("exact former inline reference-leaf default refusal", tc.input, checked, e)
			}
		}
	}
}

func TestDTOCollectionReferenceLeavesExactOldInlineGetter(t *testing.T) {
	input := schemas(`{"Number":{"type":"integer"},"Item":{"properties":{"id":{"type":"string"},"values":{"type":"array","items":{"$ref":"#/components/schemas/Number"}}}},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item"}}}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal("exact former inline getter refusal", e)
	}
}

func TestDTOCollectionReferenceLeavesLateHTTPSource(t *testing.T) {
	rq := request([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Scalar":{"type":"integer","minimum":0},"Item":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Scalar"}}}}}},"paths":{"/rows":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}},"responses":{"204":{"description":"synthetic"}}}}}}`), "Dto")
	rq.Options.Validation = true
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal("late HTTP unique source original getters", e)
	}
}

func TestDTOCollectionReferenceLeavesSourceAndCheckedBudgets(t *testing.T) {
	for _, tc := range []struct {
		aliases, depth int
		accepted       bool
	}{{0, 15, true}, {0, 16, false}, {30, 15, true}, {31, 15, false}} {
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
		defs["Item"] = map[string]any{"properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/Value"}}}
		defs["Holder"] = map[string]any{"properties": map[string]any{"values": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}}}
		data, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": defs}})
		if e != nil {
			t.Fatal(e)
		}
		rq := request(data, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		if tc.accepted && e != nil || !tc.accepted && (e == nil || len(r.Artifacts.Files) != 0) {
			t.Fatal("original source48/checked16 with leaf hop", tc, e)
		}
	}
}
