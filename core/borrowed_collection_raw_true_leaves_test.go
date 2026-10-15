package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionRawTrueLeavesOriginalJSONBacking(t *testing.T) {
	input := schemas(`{"Raw":{"properties":{},"nullable":true},"Anything":true,"Maybe":{"$ref":"#/components/schemas/Raw"},"False":{"$ref":"#/components/schemas/Maybe","nullable":false},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Maybe"}},"any":{"type":"array","items":{"$ref":"#/components/schemas/Anything"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/False"}}}},"RowsBridge":{"$ref":"#/components/schemas/Source/properties/rows"},"TrueBridge":{"$ref":"#/components/schemas/Source/properties/any"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.ModelStyle, rq.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			for _, tc := range []struct{ name, want string }{{"RowsBridgeDto", "List<kotlinx.serialization.json.JsonElement?>"}, {"TrueBridgeDto", "List<kotlinx.serialization.json.JsonElement>"}, {"MapBridgeDto", "kotlinx.serialization.json.JsonObject"}} {
				if !strings.Contains(source(t, got, tc.name), tc.want) {
					t.Fatal(tc, source(t, got, tc.name))
				}
			}
			warnings := map[string]int{}
			for _, w := range got.Warnings {
				warnings[w.Code]++
			}
			if warnings["W_JSON_ELEMENT_APPROXIMATION"] != 1 || warnings["W_BOOLEAN_SCHEMA_PROJECTION"] != 1 {
				t.Fatal("borrowed users changed source warning ownership", warnings)
			}
		}
	}
}
