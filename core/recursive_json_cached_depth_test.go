package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRecursiveJSONCachedCollectionSourceHeight(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, containers := range []int{43, 44} {
			for _, rootOrder := range []bool{false, true} {
				input := recursiveJSONChange(t, func(d map[string]any) {
					delete(d, "paths")
					schemas := d["components"].(map[string]any)["schemas"].(map[string]any)
					root := schemas["ZAny"]
					for key := range schemas {
						delete(schemas, key)
					}
					schemas["ZAny"] = root
					var collection any = map[string]any{"$ref": "#/components/schemas/ZAny"}
					for i := 0; i < containers; i++ {
						collection = map[string]any{"type": "array", "items": collection}
					}
					schemas["Rows"] = collection
					name := "AConsumer"
					if rootOrder {
						name = "ZZConsumer"
					}
					schemas[name] = map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/Rows"}}}
				})
				r := request(input, "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if containers == 43 {
					if err != nil {
						t.Fatal(checked, rootOrder, containers, err)
					}
				} else {
					if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) || !strings.Contains(err.Error(), "E_SCHEMA_DEPTH") {
						t.Fatal("cached cover source height erased", checked, rootOrder, containers, got, err)
					}
					name := "AConsumer"
					if rootOrder {
						name = "ZZConsumer"
					}
					p := fmt.Sprintf("/components/schemas/%s/properties/value", name)
					d := err.(core.Diagnostics)
					if len(d) != 1 || d[0].Pointer == nil || *d[0].Pointer != p {
						t.Fatal(d)
					}
				}
			}
		}
	}
}

func TestRecursiveJSONUnusedMalformedSource(t *testing.T) {
	for _, checked := range []bool{false, true} {
		input := recursiveJSONChange(t, func(d map[string]any) {
			delete(d, "paths")
			schemas := d["components"].(map[string]any)["schemas"].(map[string]any)
			root := schemas["ZAny"]
			for key := range schemas {
				delete(schemas, key)
			}
			schemas["ZAny"] = root
			root.(map[string]any)["oneOf"].([]any)[5].(map[string]any)["additionalProperties"].(map[string]any)["description"] = 3
		})
		var doc any
		if err := json.Unmarshal(input, &doc); err != nil {
			t.Fatal(err)
		}
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(got, err)
		}
		d := err.(core.Diagnostics)
		if len(d) != 1 || d[0].Pointer == nil || *d[0].Pointer != "/components/schemas/ZAny/oneOf/5/additionalProperties/description" {
			t.Fatal(d)
		}
	}
}
