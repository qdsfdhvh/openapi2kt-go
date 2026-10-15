package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionBigintLeavesNumericContainersFieldOnlyCodec(t *testing.T) {
	input := schemas(`{"Amount":{"x-native-type":"bigint","nullable":true},"Alias":{"$ref":"#/components/schemas/Amount"},"False":{"$ref":"#/components/schemas/Alias","nullable":false},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/False"}},"value":{"$ref":"#/components/schemas/Alias"}}},"RowsBridge":{"$ref":"#/components/schemas/Source/properties/rows"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"},"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/RowsBridge"},"map":{"$ref":"#/components/schemas/MapBridge"},"amount":{"$ref":"#/components/schemas/Alias"}}}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.ModelStyle = style
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			for _, tc := range []struct{ name, want string }{{"RowsBridgeDto", "List<kotlin.Long?>"}, {"MapBridgeDto", "Map<kotlin.String, kotlin.Long>"}} {
				if !strings.Contains(source(t, got, tc.name), tc.want) {
					t.Fatal(tc, source(t, got, tc.name))
				}
			}
			holder := source(t, got, "HolderDto")
			if strings.Count(holder, "LongAsStringSerializer::class") != 1 {
				t.Fatal("field codec leaked into numeric containers", holder)
			}
			if strings.Contains(source(t, got, "RowsBridgeDto"), "LongAsStringSerializer") || strings.Contains(source(t, got, "MapBridgeDto"), "LongAsStringSerializer") {
				t.Fatal("container source alias owns field codec")
			}
		}
	}
}
