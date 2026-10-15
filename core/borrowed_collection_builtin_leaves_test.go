package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionBuiltinLeavesCanonicalChecksAndNullable(t *testing.T) {
	doc := schemas(`{"Leaf":{"type":"string"},"Number":{"type":"integer","format":"int64"},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","nullable":true}},"map":{"additionalProperties":{"$ref":"#/components/schemas/Number"}}}},"RowsBridge":{"$ref":"#/components/schemas/Source/properties/rows"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"},"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/RowsBridge"},"map":{"$ref":"#/components/schemas/MapBridge"}},"required":["rows","map"]}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(doc, "Dto")
			rq.Options.ModelStyle = style
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			if !strings.Contains(source(t, got, "RowsBridgeDto"), "List<kotlin.String?>") || !strings.Contains(source(t, got, "MapBridgeDto"), "Map<kotlin.String, kotlin.Long>") {
				t.Fatal("borrowed named builtin backing lost")
			}
		}
	}
}
