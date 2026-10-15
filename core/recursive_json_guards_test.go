package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func recursiveJSONChange(t *testing.T, change func(map[string]any)) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(recursiveJSONInput(t), &doc); err != nil {
		t.Fatal(err)
	}
	change(doc)
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func recursiveJSONSchema(doc map[string]any) map[string]any {
	return doc["components"].(map[string]any)["schemas"].(map[string]any)["ZAny"].(map[string]any)
}
func TestRecursiveJSONSourceGuards(t *testing.T) {
	cases := []struct {
		name    string
		change  func(map[string]any)
		pointer string
	}{
		{"bound", func(d map[string]any) { recursiveJSONSchema(d)["oneOf"].([]any)[1].(map[string]any)["minLength"] = 1 }, "/components/schemas/ZAny/oneOf/1/minLength"},
		{"format", func(d map[string]any) {
			recursiveJSONSchema(d)["oneOf"].([]any)[3].(map[string]any)["format"] = "double"
		}, "/components/schemas/ZAny/oneOf/3/format"},
		{"root-nullable", func(d map[string]any) { recursiveJSONSchema(d)["nullable"] = true }, "/components/schemas/ZAny/nullable"},
		{"unknown", func(d map[string]any) { recursiveJSONSchema(d)["x-unrecognized"] = true }, "/components/schemas/ZAny/x-unrecognized"},
		{"bad-metadata", func(d map[string]any) { recursiveJSONSchema(d)["description"] = 3 }, "/components/schemas/ZAny/description"},
		{"child-nullable", func(d map[string]any) {
			recursiveJSONSchema(d)["oneOf"].([]any)[4].(map[string]any)["items"].(map[string]any)["nullable"] = false
		}, "/components/schemas/ZAny/oneOf/4/items/nullable"},
		{"bad-ref", func(d map[string]any) {
			recursiveJSONSchema(d)["oneOf"].([]any)[4].(map[string]any)["items"].(map[string]any)["$ref"] = 2
		}, "/components/schemas/ZAny/oneOf/4/items/$ref"},
		{"alias-nullable", func(d map[string]any) {
			d["components"].(map[string]any)["schemas"].(map[string]any)["AChain"].(map[string]any)["nullable"] = true
		}, "/components/schemas/AChain"},
		{"3.0", func(d map[string]any) { d["openapi"] = "3.0.3" }, "/components/schemas/ZAny/oneOf"},
		{"missing-number", func(d map[string]any) {
			b := recursiveJSONSchema(d)["oneOf"].([]any)
			recursiveJSONSchema(d)["oneOf"] = append(b[:3], b[4:]...)
		}, "/components/schemas/ZAny/oneOf"},
		{"duplicate-kind", func(d map[string]any) { recursiveJSONSchema(d)["oneOf"].([]any)[3].(map[string]any)["type"] = "string" }, "/components/schemas/ZAny/oneOf"},
	}
	for _, tc := range cases {
		for _, checked := range []bool{false, true} {
			t.Run(fmt.Sprint(tc.name, checked), func(t *testing.T) {
				r := request(recursiveJSONChange(t, tc.change), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if tc.name == "alias-nullable" {
					// Same historical input explicitly graduates to a nullable
					// canonical carrier; other source guards remain failures.
					if err != nil || !strings.Contains(source(t, got, "models/AChainDto"), "typealias AChainDto = "+r.Options.PackageName+".models.ZAnyDto?") {
						t.Fatal(got, err)
					}
					return
				}
				if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("failure must not produce a partial report", got, err)
				}
				diagnostics := err.(core.Diagnostics)
				if len(diagnostics) != 1 || diagnostics[0].Pointer == nil || *diagnostics[0].Pointer != tc.pointer {
					t.Fatal(diagnostics)
				}
			})
		}
	}
}

func TestRecursiveJSONOriginalSourceHeights(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			limit := 44 // field depth 1 + arrays + reference 1 + cover height 2
			if alias {
				limit = 46 // alias edges + cover height 2
			}
			for _, n := range []int{limit, limit + 1} {
				input := recursiveJSONChange(t, func(d map[string]any) {
					delete(d, "paths")
					schemas := d["components"].(map[string]any)["schemas"].(map[string]any)
					root := schemas["ZAny"]
					for key := range schemas {
						delete(schemas, key)
					}
					schemas["ZAny"] = root
					if alias {
						for i := 0; i < n; i++ {
							target := "ZAny"
							if i+1 < n {
								target = fmt.Sprintf("R%02d", i+1)
							}
							schemas[fmt.Sprintf("R%02d", i)] = map[string]any{"$ref": "#/components/schemas/" + target}
						}
					} else {
						var value any = map[string]any{"$ref": "#/components/schemas/ZAny"}
						for i := 0; i < n; i++ {
							value = map[string]any{"type": "array", "items": value}
						}
						schemas["Holder"] = map[string]any{"type": "object", "properties": map[string]any{"value": value}}
					}
				})
				r := request(input, "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if n == limit {
					if err != nil {
						t.Fatal(alias, n, checked, err)
					}
				} else if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) || !strings.Contains(err.Error(), "E_SCHEMA_DEPTH") {
					t.Fatal("erased source depth", alias, n, checked, got, err)
				}
			}
		}
	}
}
