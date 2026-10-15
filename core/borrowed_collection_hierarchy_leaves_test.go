package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionHierarchyLeavesCanonicalParentsLeavesNullAliases(t *testing.T) {
	doc := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Group"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind","mapping":{"group":"#/components/schemas/Group","other":"#/components/schemas/Other"}}},"Group":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"sub","mapping":{"first":"#/components/schemas/First","second":"#/components/schemas/Second"}}},"First":{"properties":{"id":{"type":"string"}},"required":["id"]},"Second":{"properties":{"name":{"type":"string"}},"required":["name"]},"Other":{"properties":{"value":{"type":"string"}},"required":["value"]},"Maybe":{"$ref":"#/components/schemas/Root","nullable":true},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Root"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/First","nullable":true}},"nullable":{"type":"array","items":{"$ref":"#/components/schemas/Maybe"}}}},"RowsBridge":{"$ref":"#/components/schemas/Source/properties/rows"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"},"MaybeBridge":{"$ref":"#/components/schemas/Source/properties/nullable"}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(doc, "Dto")
			rq.Options.ModelStyle = style
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			for _, tc := range []struct{ name, want string }{{"RowsBridgeDto", "List<org.example.models.models.RootDto>"}, {"MapBridgeDto", "Map<kotlin.String, org.example.models.models.FirstDto?>"}, {"MaybeBridgeDto", "List<org.example.models.models.RootDto?>"}} {
				if !strings.Contains(source(t, got, tc.name), tc.want) {
					t.Fatal(tc, source(t, got, tc.name))
				}
			}
		}
	}
}
