package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionCallerNativeLeavesProviderOwnedBacking(t *testing.T) {
	input := schemas(`{"Money":{"x-native-type":"org.example.scalarvalues.Money","nullable":true},"NativeLong":{"x-native-type":"kotlin.Long"},"Alias":{"$ref":"#/components/schemas/Money"},"False":{"$ref":"#/components/schemas/Alias","nullable":false},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/False"}},"longs":{"type":"array","items":{"$ref":"#/components/schemas/NativeLong"}}}},"RowsBridge":{"$ref":"#/components/schemas/Source/properties/rows"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"},"LongsBridge":{"$ref":"#/components/schemas/Source/properties/longs"}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.ModelStyle, rq.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			for _, tc := range []struct{ name, want string }{{"RowsBridgeDto", "List<org.example.scalarvalues.Money?>"}, {"MapBridgeDto", "Map<kotlin.String, org.example.scalarvalues.Money>"}, {"LongsBridgeDto", "List<kotlin.Long>"}} {
				if !strings.Contains(source(t, got, tc.name), tc.want) {
					t.Fatal(tc, source(t, got, tc.name))
				}
			}
			for _, file := range got.Artifacts.Files {
				if strings.Contains(file.Content, "LongAsStringSerializer") {
					t.Fatal("caller Long acquired magic bigint helper", file.Path)
				}
			}
		}
	}
}
