package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionBytesLeavesNumericArrayContainersNotRawLeaf(t *testing.T) {
	input := schemas(`{"Blob":{"type":"string","format":"byte","nullable":true},"Binary":{"type":"string","format":"binary"},"Alias":{"$ref":"#/components/schemas/Blob"},"False":{"$ref":"#/components/schemas/Alias","nullable":false},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/Binary","nullable":true}},"narrow":{"type":"array","items":{"$ref":"#/components/schemas/False"}}}},"RowsBridge":{"$ref":"#/components/schemas/Source/properties/rows"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"},"NarrowBridge":{"$ref":"#/components/schemas/Source/properties/narrow"}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.ModelStyle = style
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			for _, tc := range []struct{ name, want string }{{"RowsBridgeDto", "List<kotlin.ByteArray?>"}, {"MapBridgeDto", "Map<kotlin.String, kotlin.ByteArray?>"}, {"NarrowBridgeDto", "List<kotlin.ByteArray>"}} {
				if !strings.Contains(source(t, got, tc.name), tc.want) {
					t.Fatal(tc, source(t, got, tc.name))
				}
			}
		}
	}
}
