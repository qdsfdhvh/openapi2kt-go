package core_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRecursiveJSONKindsAndCatalogOrder(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(recursiveJSONInput(t), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			original, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			// Keep Holder property RawMessages intact; only catalog inventory and
			// disjoint root branches are reordered, not constructor parameters.
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
			var cover map[string]json.RawMessage
			if err := json.Unmarshal(schemas["ZAny"], &cover); err != nil {
				t.Fatal(err)
			}
			var branches []json.RawMessage
			if err := json.Unmarshal(cover["oneOf"], &branches); err != nil {
				t.Fatal(err)
			}
			for i, j := 0, len(branches)-1; i < j; i, j = i+1, j-1 {
				branches[i], branches[j] = branches[j], branches[i]
			}
			delete(cover, "oneOf")
			cover["anyOf"], _ = json.Marshal(branches)
			schemas["ZAny"], _ = json.Marshal(cover)
			var b bytes.Buffer
			b.WriteByte('{')
			for i, name := range []string{"ZAny", "Middle", "Holder", "AChain"} {
				if i != 0 {
					b.WriteByte(',')
				}
				key, _ := json.Marshal(name)
				b.Write(key)
				b.WriteByte(':')
				b.Write(schemas[name])
			}
			b.WriteByte('}')
			components["schemas"] = b.Bytes()
			doc["components"], _ = json.Marshal(components)
			r.Document, _ = json.Marshal(doc)
			permuted, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(original, permuted) {
				t.Fatal("kind/catalog order changed carrier", checked, style, err)
			}
		}
	}
}

func TestRecursiveJSONEscapedOwner(t *testing.T) {
	input := strings.ReplaceAll(string(recursiveJSONInput(t)), "#/components/schemas/ZAny", "#/components/schemas/Z~1Any~0Root")
	input = strings.Replace(input, `"ZAny": {`, `"Z/Any~Root": {`, 1)
	for _, checked := range []bool{false, true} {
		r := request([]byte(input), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Warnings) != 1 || got.Warnings[0].Pointer == nil || *got.Warnings[0].Pointer != "/components/schemas/Z~1Any~0Root" {
			t.Fatal(got.Warnings)
		}
		if !strings.Contains(source(t, got, "models/ZAnyRootDto"), "typealias ZAnyRootDto = kotlinx.serialization.json.JsonElement") {
			t.Fatal(got)
		}
	}
}

func TestRecursiveJSONNotShapeOrInternalProjection(t *testing.T) {
	for _, use := range []map[string]any{
		{"allOf": []any{map[string]any{"$ref": "#/components/schemas/ZAny"}}},
		{"oneOf": []any{map[string]any{"$ref": "#/components/schemas/ZAny"}, map[string]any{"type": "string"}}},
		{"type": "object", "properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/ZAny/oneOf/4/items"}}},
	} {
		for _, checked := range []bool{false, true} {
			input := recursiveJSONChange(t, func(d map[string]any) { d["components"].(map[string]any)["schemas"].(map[string]any)["Consumer"] = use })
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if _, typed := use["properties"]; typed {
				// Keep the original pointer-to-ref leaf input. A complete named
				// cover now grants transparent typed backing with its SCC height;
				// this never permits borrowing nominal/composed shape.
				if err != nil || !strings.Contains(source(t, got, "ConsumerDto"), "value: kotlinx.serialization.json.JsonElement? = null") {
					t.Fatal("original cover typed-chain graduation", checked, err)
				}
				continue
			}
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("JSON cover cannot lend nominal shape/internal cycle", use, checked, got, err)
			}
		}
	}
}
