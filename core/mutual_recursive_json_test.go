package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func mutualJSONInput(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/mutual-recursive-json.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestMutualRecursiveJSONWholeEnvelope(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(mutualJSONInput(t), "Dto")
			r.Options.ModelStyle, r.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			for _, name := range []string{"ZAny", "ZOther"} {
				if !strings.Contains(source(t, got, "models/"+name+"Dto"), "typealias "+name+"Dto = kotlinx.serialization.json.JsonElement") {
					t.Fatal(got)
				}
			}
			// Preserve constructor/property declaration order in the control.
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
			control, _ := json.Marshal(doc)
			r.Document = control
			before, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got.Artifacts.Files, before.Artifacts.Files) {
				t.Fatal("carrier changed", err)
			}
			if len(got.Warnings) != 2 {
				t.Fatal(got.Warnings)
			}
			for i, name := range []string{"ZAny", "ZOther"} {
				w := got.Warnings[i]
				if w.Code != "W_RECURSIVE_JSON_PROJECTION" || w.Pointer == nil || *w.Pointer != "/components/schemas/"+name {
					t.Fatal(w)
				}
			}
			r.Document = mutualJSONInput(t)
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic", err)
			}
		}
	}
}

func mutualJSONChange(t *testing.T, change func(map[string]any)) []byte {
	t.Helper()
	var d map[string]any
	if err := json.Unmarshal(mutualJSONInput(t), &d); err != nil {
		t.Fatal(err)
	}
	change(d["components"].(map[string]any)["schemas"].(map[string]any))
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestMutualRecursiveJSONSourceAndConsumerDepth(t *testing.T) {
	for _, checked := range []bool{false, true} {
		// Two covers contribute four original edges, including the closing
		// reference. A cached List alias must retain the complete source work.
		for _, containers := range []int{41, 42} {
			input := mutualJSONChange(t, func(s map[string]any) {
				var value any = map[string]any{"$ref": "#/components/schemas/ZAny"}
				for i := 0; i < containers; i++ {
					value = map[string]any{"type": "array", "items": value}
				}
				s["Rows"] = value
				s["Deep"] = map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/Rows"}}}
			})
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if containers == 41 {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) || !strings.Contains(err.Error(), "E_SCHEMA_DEPTH") {
				t.Fatal(got, err)
			}
		}
	}
}

func TestMutualRecursiveJSONNoDrop(t *testing.T) {
	cases := []struct {
		name   string
		change func(map[string]any)
	}{
		{"other-bound", func(s map[string]any) {
			s["ZOther"].(map[string]any)["anyOf"].([]any)[3].(map[string]any)["minimum"] = 1
		}},
		{"partial", func(s map[string]any) { v := s["ZOther"].(map[string]any); v["anyOf"] = v["anyOf"].([]any)[:5] }},
		{"not-json", func(s map[string]any) {
			s["ZOther"] = map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}
		}},
		{"transparent-cycle", func(s map[string]any) { s["ZOther"] = map[string]any{"$ref": "#/components/schemas/ZOther"} }},
	}
	for _, tc := range cases {
		for _, checked := range []bool{false, true} {
			t.Run(fmt.Sprint(tc.name, checked), func(t *testing.T) {
				r := request(mutualJSONChange(t, tc.change), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("partial graph erased", got, err)
				}
			})
		}
	}
}
