package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionTemporalLeavesNativeISOBackingAndNullable(t *testing.T) {
	doc := schemas(`{"Time":{"type":"string","format":"date-time","nullable":true},"Date":{"type":"string","format":"date"},"NativeDate":{"type":"string","x-native-type":"date"},"MaybeTime":{"$ref":"#/components/schemas/Time"},"FalseTime":{"$ref":"#/components/schemas/MaybeTime","nullable":false},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/MaybeTime"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/Date","nullable":true}},"narrow":{"type":"array","items":{"$ref":"#/components/schemas/FalseTime"}},"native":{"type":"array","items":{"$ref":"#/components/schemas/NativeDate"}}}},"RowsBridge":{"$ref":"#/components/schemas/Source/properties/rows"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"},"NarrowBridge":{"$ref":"#/components/schemas/Source/properties/narrow"}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(doc, "Dto")
			rq.Options.ModelStyle = style
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			for _, tc := range []struct{ name, want string }{{"RowsBridgeDto", "List<kotlin.time.Instant?>"}, {"MapBridgeDto", "Map<kotlin.String, kotlinx.datetime.LocalDate?>"}, {"NarrowBridgeDto", "List<kotlin.time.Instant>"}} {
				if !strings.Contains(source(t, got, tc.name), tc.want) {
					t.Fatal(tc, source(t, got, tc.name))
				}
			}
		}
	}
}
