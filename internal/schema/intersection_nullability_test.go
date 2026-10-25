package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestIntersectionOriginalShapesNullViewsAndAliases(t *testing.T) {
	input, err := os.ReadFile("../../tests/fixtures/http/nullable-product-unions.json")
	if err != nil {
		t.Fatal(err)
	}
	root, failure := document.Load(input, document.JSON, "synthetic.json")
	if failure != nil {
		t.Fatal(failure)
	}
	before, _ := root.MarshalJSON()
	a := &analyzer{root: root, options: Options{PackageName: "org.example.intersections", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	declarations, failure := a.analyze(root)
	if failure != nil {
		t.Fatal(failure)
	}
	catalog := get(get(root, "components"), "schemas")
	if len(a.components) != len(catalog.Members()) {
		t.Fatal("intersection created synthetic catalog identities")
	}
	pair := a.components["JointPair"]
	if !pair.NullableDTO || pair.Schema != get(catalog, "JointPair") {
		t.Fatal("derived null intent lost original component")
	}
	var actual *Model
	for _, d := range declarations {
		if m, ok := d.(*Model); ok && m.Name() == "JointPairDto" {
			actual = m
		}
	}
	if actual == nil || !actual.Nullable || a.modelOrigins[actual] != pair.Schema || len(actual.Properties) != 3 {
		t.Fatal("physical constructor or source was replaced")
	}
	shape, failure := a.objectShape(pair.Schema, pair.Pointer, 0)
	left := get(get(get(catalog, "JointA"), "properties"), "left")
	if failure != nil || shapeField(shape, "left").Schema != left || shapeField(shape, "left").Pointer != "/components/schemas/JointA/properties/left" {
		t.Fatal("member fields did not borrow original sources", failure)
	}
	for _, name := range []string{"JointProductTT", "JointProductTF", "JointProductFT", "JointProductFF"} {
		c := a.components[name]
		view, failure := a.objectCompositionUnion(c.Schema, c.Pointer, 0)
		if failure != nil || view == nil || view.Nullable != (name == "JointProductTT") {
			t.Fatal("product AND null view", name, view, failure)
		}
		for _, branch := range view.Branches {
			if branch.Shape.Nullable != nil && *branch.Shape.Nullable {
				t.Fatal("nullable view leaked into physical alternative", name)
			}
		}
	}
	for _, path := range []string{"JointNestedAlias", "JointNestedFalse"} {
		target, failure := a.resolveSchema("#/components/schemas/"+path, "/use")
		if failure != nil {
			t.Fatal(failure)
		}
		ty, failure := a.referenceType(target, nil, 0, "/use")
		if failure != nil || (ty.Kind == Nullable) != (path == "JointNestedAlias") {
			t.Fatal("derived internal product alias null intent", path, ty, failure)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("intersection rewrote original source nodes")
	}
}
