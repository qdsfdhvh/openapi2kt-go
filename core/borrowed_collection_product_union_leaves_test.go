package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func borrowedProductUnionSource() string {
	return `{"allOf":[{"oneOf":[{"properties":{"kind":{"const":"id"},"id":{"type":"string"}},"required":["id"]},{"properties":{"kind":{"const":"name"},"name":{"type":"string"}},"required":["name"]}]},{"properties":{"common":{"type":"string"}}}]}`
}

func TestBorrowedCollectionProductUnionLeavesCanonicalAPI(t *testing.T) {
	product := borrowedProductUnionSource()
	doc := schemas(`{"RowsBridge":{"$ref":"#/components/schemas/Source/properties/rows"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/ProductAlias"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/Owner/properties/choice","nullable":true}}}},"ProductAlias":{"$ref":"#/components/schemas/Product"},"Product":` + product + `,"Owner":{"properties":{"choice":` + product + `}},"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/RowsBridge"},"map":{"$ref":"#/components/schemas/MapBridge"}}}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(doc, "Dto")
			rq.Options.ModelStyle, rq.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			for _, tc := range []struct{ name, backing string }{
				{"RowsBridgeDto", "List<org.example.models.models.ProductDto>"},
				{"MapBridgeDto", "Map<kotlin.String, org.example.models.models.OwnerDto.Choice?>"},
			} {
				if !strings.Contains(source(t, got, tc.name), tc.backing) {
					t.Fatal("canonical collection product alias missing", tc, source(t, got, tc.name))
				}
			}
			productSource := source(t, got, "ProductDto")
			for _, want := range []string{"sealed class ProductDto", "class Id(", "class Name(", "UnknownProductDto"} {
				if !strings.Contains(productSource, want) {
					t.Fatal("original product API lost", want, productSource)
				}
			}
		}
	}
}

func borrowedProductGraduation(t *testing.T, input []byte, canonical string) {
	t.Helper()
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("original product collection input did not graduate", err)
		}
		all := ""
		for _, file := range got.Artifacts.Files {
			all += file.Content
		}
		if !strings.Contains(all, "kotlin.collections.List<"+canonical+">") || !strings.Contains(all, "sealed class") {
			t.Fatal("original product classifier lost", canonical)
		}
	}
}
