package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestReferencedStringEnumCollectionFieldDefaultsSourceAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"AHolder":{"properties":{"v":{"$ref":"#/components/schemas/Hops/properties/narrow","default":[["on"],null,[]]}}},"Hops":{"properties":{"entry":{"$ref":"#/components/schemas/Rows","nullable":true},"narrow":{"$ref":"#/components/schemas/Hops/properties/entry","nullable":false}}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/NullableRows"}},"NullableRows":{"type":["array","null"],"items":{"type":["string","null"],"enum":["on","off"]}},"Status":{"type":"string","enum":["on"]}}}}`)
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
	if len(pending) != 1 {
		t.Fatal(len(pending))
	}
	p := pending[0]
	c := a.components["Rows"]
	ty := p.Type.WithoutRootNull()
	if ty.Kind != Named || ty.Element != &c.AliasType || ty.ArrayAliasView != NonNullArrayAlias {
		t.Fatal("actual alias/false backing", ty)
	}
	node, err := a.referencedDefaultSchema(p.Field.Schema, p.Field.Pointer, p.Type, p.Field.Depth, true)
	if err != nil {
		t.Fatal(err)
	}
	backing := c.AliasType.WithoutRootNull()
	rows := a.components["NullableRows"]
	leaf := node.element.element
	var enum *StringEnum
	for _, d := range ds {
		if e, ok := d.(*StringEnum); ok && e.ClassName == "NullableRowsDtoInner" {
			enum = e
		}
	}
	if node.schema != c.Schema || node.ty.Kind != List || node.ty.Collection != backing.Collection || node.element.schema != rows.Schema || node.element.ty.WithoutRootNull().Kind != List || leaf.schema != get(rows.Schema, "items") || !leaf.stringEnum || leaf.ty.WithoutRootNull().Kind != Named || enum == nil || len(enum.Members) != 2 || enum.Members[0].Wire != "on" || enum.Members[1].Wire != "off" {
		t.Fatal("physical source/declaration/vocabulary identity lost", node, enum)
	}
	target, err := a.resolveSchema("#/components/schemas/Hops/properties/narrow", "/use")
	if err != nil || target.CanonicalCollection != c || target.Steps != 3 {
		t.Fatal("cached source height", target, err)
	}
	// A Named type spelling alone cannot supply the source role.
	if _, err := a.referencedDefaultSchemaRole(leaf.schema, leaf.pointer, leaf.ty, 0, false, noReferencedDefaultRole); err == nil {
		t.Fatal("ungranted coincident enum authorized")
	}
	// The plain scalar terminal path must not swallow enum terminals.
	if _, err := a.referencedDefaultSchemaRole(leaf.schema, leaf.pointer, TypeRef{Kind: Scalar, Name: "kotlin.String"}, 0, false, referencedCollectionOwnDefault); err == nil {
		t.Fatal("coincident scalar authorized enum terminal")
	}
	for _, tc := range []struct {
		name        string
		replacement func() Declaration
	}{{c.Name, func() Declaration { return &ValueAlias{ClassName: c.Name, Type: c.AliasType} }}} {
		without := []Declaration{}
		var original Declaration
		for _, d := range ds {
			if d.Name() == tc.name {
				original = d
			} else {
				without = append(without, d)
			}
		}
		if original == nil {
			t.Fatal("missing original export", tc.name)
		}
		if a.validateInternalModels(without) == nil {
			t.Fatal("missing declaration authorized", tc.name)
		}
		replacement := tc.replacement()
		if a.validateInternalModels(append(append([]Declaration{}, without...), replacement)) == nil {
			t.Fatal("same-name copied declaration authorized", tc.name)
		}
		if a.validateInternalModels(append(append([]Declaration{}, ds...), original)) == nil {
			t.Fatal("duplicate original declaration authorized", tc.name)
		}
	}
	for _, owner := range []*component{c} {
		saved := owner.Schema
		b, _ := saved.MarshalJSON()
		copy, err := document.Load(b, document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		owner.Schema = copy
		if a.validateInternalModels(ds) == nil {
			t.Fatal("copied physical source authorized")
		}
		owner.Schema = saved
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("immutable source tree changed")
	}
}
