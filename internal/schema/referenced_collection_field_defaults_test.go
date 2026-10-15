package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestReferencedCollectionFieldDefaultsKeepOriginalAliasAndChecks(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Hops/properties/rows","default":[1]},"again":{"$ref":"#/components/schemas/Hops/properties/rows","default":[2]}}},"Hops":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}},"Rows":{"type":"array","minItems":1,"uniqueItems":true,"items":{"type":"integer","minimum":0}}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	if err := a.planReferencedFieldDefaults(); err != nil {
		t.Fatal(err)
	}
	c := a.components["Rows"]
	backing := c.AliasType.WithoutRootNull()
	if backing.Collection == nil || !backing.Collection.Unique || backing.Collection.Minimum == nil || backing.Element.Validation == nil {
		t.Fatal("original checks missing")
	}
	for _, plan := range pending {
		ty := plan.Type.WithoutRootNull()
		if ty.Kind != Named || ty.Element != &c.AliasType || ty.Name != "org.example.defaults.models.RowsDto" {
			t.Fatal("copied original TypeRef backing", ty)
		}
		node, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
		if err != nil || node.schema != c.Schema || node.ty.Collection != backing.Collection || node.element.schema != get(c.Schema, "items") || node.element.ty.Validation != backing.Element.Validation {
			t.Fatal("lost source/check identity", err)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source mutated")
	}
	// The original actual declaration is still the authority, even with defaults.
	for _, d := range ds {
		if d.Name() == c.Name {
			missing := []Declaration{}
			for _, candidate := range ds {
				if candidate != d {
					missing = append(missing, candidate)
				}
			}
			if err := a.validateNamedCollectionBindings(missing); err == nil {
				t.Fatal("missing actual alias admitted")
			}
			if err := a.validateNamedCollectionBindings(append(append([]Declaration{}, ds...), d)); err == nil {
				t.Fatal("duplicate actual alias admitted")
			}
		}
	}
	saved := *backing.Collection.Minimum
	*backing.Collection.Minimum++
	if err := a.validateNamedCollectionBindings(ds); err == nil {
		t.Fatal("mutated source checks admitted")
	}
	*backing.Collection.Minimum = saved
	if err := a.validateNamedCollectionBindings(ds); err != nil {
		t.Fatal(err)
	}
}
