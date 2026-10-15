package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func composedUnionTerminalsInput(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/composed-union-terminals.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestComposedUnionActualTerminalsAndNullViews(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			req := request(composedUnionTerminalsInput(t), "Dto")
			req.Options.ModelStyle, req.Options.Validation = style, checked
			r, err := core.GenerateWithDiagnostics(req)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			for _, tc := range []struct {
				alias, owner string
				nullable     bool
			}{
				{"ProductAlias", "ProductDto", false}, {"ProductMaybe", "ProductDto", true}, {"ProductFalse", "ProductDto", false},
				{"NestedAlias", "BoxDto.Choice", false}, {"NestedMaybe", "BoxDto.Choice", true}, {"NestedFalse", "BoxDto.Choice", false},
				{"RowAlias", "RowsDtoInner", false}, {"MapAlias", "MapDtoValue", false}, {"RootBridge", "ProductDto", false}, {"NestedBridge", "BoxDto.Choice", false}, {"NarrowBridge", "ProductDto", false},
			} {
				want := "public typealias " + tc.alias + "Dto = org.example.models.models." + tc.owner
				if tc.nullable {
					want += "?"
				}
				if !strings.Contains(source(t, r, tc.alias+"Dto"), want+"\n") {
					t.Fatal(tc, source(t, r, tc.alias+"Dto"))
				}
			}
			product := source(t, r, "ProductDto")
			for _, part := range []string{"public sealed class ProductDto", "class Note(", "class Note2(", "class Count(", "class Count4(", "UnknownProductDto"} {
				if !strings.Contains(product, part) {
					t.Fatal(part, product)
				}
			}
			patch := source(t, r, "PatchProductInputDto")
			if !strings.Contains(patch, "Patch<org.example.models.models.BoxDto.Choice>") {
				t.Fatal(patch)
			}
			again, err := core.GenerateWithDiagnostics(req)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("nondeterministic", err)
			}
		}
	}
}

func TestComposedUnionStandaloneAndNestedReferenceRegressions(t *testing.T) {
	product := `{"allOf":[{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"id":{"type":"string"}},"required":["id"]},{"properties":{"kind":{"type":"string","const":"b"},"name":{"type":"string"}},"required":["name"]}]},{"properties":{"common":{"type":"string"}}}]}`
	for _, body := range []string{
		`"Owner":` + product + `,"Hops":{"properties":{"hop":{"$ref":"#/components/schemas/Owner"}}},"Alias":{"$ref":"#/components/schemas/Hops/properties/hop"}`,
		`"Owner":{"properties":{"choice":` + product + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/choice"}`,
		`"Owner":` + product + `,"Alias":{"$ref":"#/components/schemas/Owner","nullable":true}`,
	} {
		for _, checked := range []bool{false, true} {
			req := request(schemas(`{`+body+`}`), "Dto")
			req.Options.Validation = checked
			r, err := core.GenerateWithDiagnostics(req)
			if err != nil {
				t.Fatal(body, checked, err)
			}
			if !strings.Contains(source(t, r, "AliasDto"), "public typealias AliasDto = org.example.models.models.OwnerDto") {
				t.Fatal(r)
			}
		}
	}
}
