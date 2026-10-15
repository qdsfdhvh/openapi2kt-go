package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestNullableProductUnionsBorrowOriginalPhysicalShapes(t *testing.T) {
	input, err := os.ReadFile("../../tests/fixtures/http/nullable-product-unions.json")
	if err != nil {
		t.Fatal(err)
	}
	root, e := document.Load(input, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.products", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	catalog := get(get(root, "components"), "schemas")
	product := get(catalog, "Product")
	view, e := a.objectCompositionUnion(product, "/components/schemas/Product", 0)
	if e != nil || view == nil || !view.Nullable || len(view.Branches) != 2 {
		t.Fatal(view, e)
	}
	id := get(get(get(get(product, "allOf").Items()[0], "oneOf").Items()[0], "properties"), "id")
	if field := shapeField(view.Branches[0].Shape, "id"); field.Schema != id || field.Pointer != "/components/schemas/Product/allOf/0/oneOf/0/properties/id" {
		t.Fatal("original product source replaced", field)
	}
	for _, branch := range view.Branches {
		if branch.Shape.Nullable != nil && *branch.Shape.Nullable {
			t.Fatal("root nullable intent leaked into physical variant", branch)
		}
	}
	physical := 0
	for _, d := range ds {
		if u, ok := d.(*ObjectUnion); ok && u.Name() == "ProductDto" {
			physical++
			if !u.NullVariant || a.unionOrigins[u].Schema != product || a.unionOrigins[u].Nullable || a.components["Product"].NullableDTO {
				t.Fatal("root carrier acquired nullable classifier", u, a.unionOrigins[u])
			}
		}
	}
	if physical != 1 || len(a.components) != len(catalog.Members()) {
		t.Fatal("product identity or catalog was fabricated")
	}
	for _, path := range []string{"Box/properties/value", "NestedAlias", "NestedFalse"} {
		target, e := a.resolveSchema("#/components/schemas/"+path, "/use")
		if e != nil {
			t.Fatal(path, e)
		}
		ty, e := a.referenceType(target, nil, 0, "/use")
		if e != nil || (ty.Kind == Nullable) != (path != "NestedFalse") {
			t.Fatal("original nested product null view", path, ty, e)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source graph was rewritten")
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d.Name() != "BoxDto" {
			missing = append(missing, d)
		}
	}
	if e := a.validateInternalModels(missing); e == nil || e.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("missing physical nested product authorized", e)
	}
}
