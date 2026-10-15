package core_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func recursiveRefGraphInput(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/recursive-json-reference-graph.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRecursiveJSONReferenceGraph(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(recursiveRefGraphInput(t), "Dto")
			r.Options.ModelStyle, r.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			checkBooleanRootNullableTypes(t, r, got)
			var doc, components, schemas map[string]json.RawMessage
			if err := json.Unmarshal(r.Document, &doc); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(doc["components"], &components); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(components["schemas"], &schemas); err != nil {
				t.Fatal(err)
			}
			schemas["ZAny"], schemas["ZOther"] = json.RawMessage("true"), json.RawMessage("true")
			components["schemas"], _ = json.Marshal(schemas)
			doc["components"], _ = json.Marshal(components)
			control := r
			control.Document, _ = json.Marshal(doc)
			before, err := core.GenerateWithDiagnostics(control)
			if err != nil || !reflect.DeepEqual(got.Artifacts.Files, before.Artifacts.Files) {
				t.Fatal("nullable carrier/control changed", err)
			}
			if len(got.Warnings) != 2 {
				t.Fatal(got.Warnings)
			}
			for _, name := range []string{"ZAny", "ZOther"} {
				if !strings.Contains(source(t, got, "models/"+name+"Dto"), "typealias "+name+"Dto = kotlinx.serialization.json.JsonElement") {
					t.Fatal(name, got)
				}
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("non-deterministic", err)
			}
		}
	}
}

func TestRecursiveJSONReferenceGraphPureCycleRejected(t *testing.T) {
	var d map[string]any
	if err := json.Unmarshal(recursiveRefGraphInput(t), &d); err != nil {
		t.Fatal(err)
	}
	s := d["components"].(map[string]any)["schemas"].(map[string]any)
	s["Through"] = map[string]any{"$ref": "#/components/schemas/Back"}
	s["Back"] = map[string]any{"$ref": "#/components/schemas/Through"}
	b, _ := json.Marshal(d)
	for _, checked := range []bool{false, true} {
		r := request(b, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("pure ref cycle was projected", got, err)
		}
	}
}

func TestRecursiveJSONNullableAliasSourceDepth(t *testing.T) {
	for _, n := range []int{43, 44} {
		var v any = map[string]any{"$ref": "#/components/schemas/Nullable", "nullable": false}
		for i := 0; i < n; i++ {
			v = map[string]any{"type": "array", "items": v}
		}
		var d map[string]any
		if err := json.Unmarshal(recursiveRefGraphInput(t), &d); err != nil {
			t.Fatal(err)
		}
		delete(d, "paths")
		s := d["components"].(map[string]any)["schemas"].(map[string]any)
		// Direct self-cover height2 + nullable alias1 + ref1 + field1.
		root := s["ZAny"].(map[string]any)
		branches := root["oneOf"].([]any)
		branches[4].(map[string]any)["items"] = map[string]any{"$ref": "#/components/schemas/ZAny"}
		branches[5].(map[string]any)["additionalProperties"] = map[string]any{"$ref": "#/components/schemas/ZAny"}
		for k := range s {
			if k != "ZAny" {
				delete(s, k)
			}
		}
		s["Nullable"] = map[string]any{"$ref": "#/components/schemas/ZAny", "nullable": true}
		s["Holder"] = map[string]any{"type": "object", "properties": map[string]any{"value": v}}
		b, _ := json.Marshal(d)
		r := request(b, "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if n == 43 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) || !strings.Contains(err.Error(), "E_SCHEMA_DEPTH") {
			t.Fatal(got, err)
		}
	}
}
