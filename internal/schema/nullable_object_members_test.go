package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestNullableObjectMembersKeepOriginalNodesAndPhysicalVariants(t *testing.T) {
	input, err := os.ReadFile("../../tests/fixtures/http/nullable-object-members.json")
	if err != nil {
		t.Fatal(err)
	}
	root, e := document.Load(input, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.members", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	catalog := get(get(root, "components"), "schemas")
	product := get(catalog, "Product")
	view, e := a.readUnion(product, "/components/schemas/Product")
	if e != nil || view == nil || !view.Nullable || view.HasNull || len(view.Branches) != 2 {
		t.Fatal(view, e)
	}
	member := get(product, "oneOf").Items()[0]
	if view.Branches[0].Schema != member {
		t.Fatal("nullable source branch copied")
	}
	count := 0
	for _, d := range ds {
		if u, ok := d.(*ObjectUnion); ok && u.Name() == "ProductDto" {
			count++
			if !u.NullVariant || a.unionOrigins[u].Nullable || a.unionOrigins[u].Schema != product || a.modelOrigins[u.Variants[0]] != member {
				t.Fatal("physical source classifier changed", u, a.unionOrigins[u])
			}
			id := get(get(member, "properties"), "id")
			if a.modelPointers[u.Variants[0]] != "/components/schemas/Product/oneOf/0" || id == nil {
				t.Fatal("original model source lost")
			}
		}
	}
	if count != 1 || len(a.components) != len(catalog.Members()) {
		t.Fatal("original catalog/classifier not retained")
	}
	for _, path := range []string{"Box/properties/value", "NestedAlias", "NestedFalse"} {
		target, e := a.resolveSchema("#/components/schemas/"+path, "/use")
		if e != nil {
			t.Fatal(path, e)
		}
		ty, e := a.referenceType(target, nil, 0, "/use")
		if e != nil || (ty.Kind == Nullable) != (path != "NestedFalse") {
			t.Fatal("member-derived nested null view", path, ty, e)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("member null intent rewrote source graph")
	}
}
