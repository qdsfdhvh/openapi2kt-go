package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionRecursiveCoverLeavesFullGraphJSONBacking(t *testing.T) {
	for _, catalog := range []string{`"Value":` + coverChainRoot("Value", "Value"), `"Value":` + coverChainRoot("ToOther", "Other") + `,"ToOther":{"$ref":"#/components/schemas/Other"},"Other":` + coverChainRoot("ToValue", "Value") + `,"ToValue":{"$ref":"#/components/schemas/Value"}`} {
		input := schemas(`{` + catalog + `,"Maybe":{"$ref":"#/components/schemas/Value","nullable":true},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Maybe"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/Value"}}}},"RowsBridge":{"$ref":"#/components/schemas/Source/properties/rows"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"}}`)
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				rq := request(input, "Dto")
				rq.Options.ModelStyle, rq.Options.Validation = style, checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(style, checked, err)
				}
				for _, tc := range []struct{ name, want string }{{"RowsBridgeDto", "List<kotlinx.serialization.json.JsonElement?>"}, {"MapBridgeDto", "kotlinx.serialization.json.JsonObject"}} {
					if !strings.Contains(source(t, got, tc.name), tc.want) {
						t.Fatal(tc, source(t, got, tc.name))
					}
				}
				count := 0
				for _, w := range got.Warnings {
					if w.Code == "W_RECURSIVE_JSON_PROJECTION" {
						count++
					}
				}
				want := 1
				if strings.Contains(catalog, `"Other":`) {
					want = 2
				}
				if count != want {
					t.Fatal("original graph warnings duplicated/lost", count, want)
				}
			}
		}
	}
}
