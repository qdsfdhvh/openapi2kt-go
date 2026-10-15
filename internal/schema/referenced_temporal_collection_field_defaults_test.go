package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestReferencedTemporalCollectionFieldDefaultsSourceAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"AHolder":{"properties":{"v":{"$ref":"#/components/schemas/Hops/properties/narrow","default":[{"a":"2024-02-29T12:34:56.123456789+02:30"}]}}},"Hops":{"properties":{"entry":{"$ref":"#/components/schemas/Rows","nullable":true},"narrow":{"$ref":"#/components/schemas/Hops/properties/entry","nullable":false},"time":{"$ref":"#/components/schemas/Stamp"}}},"Rows":{"type":"array","nullable":true,"minItems":1,"items":{"additionalProperties":{"$ref":"#/components/schemas/Hops/properties/time","nullable":false},"maxProperties":1}},"Stamp":{"type":"string","format":"date-time","nullable":true}}}}`)
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
	stamp := a.components["Stamp"]
	ty := p.Type.WithoutRootNull()
	if ty.Kind != Named || ty.Element != &c.AliasType || ty.ArrayAliasView != NonNullArrayAlias {
		t.Fatal("actual alias/false backing", ty)
	}
	node, err := a.referencedDefaultSchema(p.Field.Schema, p.Field.Pointer, p.Type, p.Field.Depth, true)
	if err != nil {
		t.Fatal(err)
	}
	backing := c.AliasType.WithoutRootNull()
	if node.schema != c.Schema || node.ty.Kind != List || node.ty.Collection != backing.Collection || node.element.schema != get(c.Schema, "items") || node.element.ty.Collection != backing.Element.Collection || node.element.element.schema != stamp.Schema || node.element.element.pointer != stamp.Pointer || node.element.element.ty.Kind != Scalar || node.element.element.ty.Name != "kotlin.time.Instant" {
		t.Fatal("physical source/check/nullable identity lost", node)
	}
	target, err := a.resolveSchema("#/components/schemas/Hops/properties/narrow", "/use")
	if err != nil || target.CanonicalCollection != c || target.Steps != 3 {
		t.Fatal("cached source height", target, err)
	}
	// An actual scalar type spelling alone cannot supply a source role.
	if _, err := a.referencedDefaultSchemaRole(stamp.Schema, stamp.Pointer, TypeRef{Kind: Scalar, Name: "kotlin.time.Instant"}, 0, false, noReferencedDefaultRole); err == nil {
		t.Fatal("ungranted coincident scalar authorized")
	}
	for _, name := range []string{c.Name, stamp.Name} {
		without := []Declaration{}
		var original Declaration
		for _, d := range ds {
			if d.Name() == name {
				original = d
			} else {
				without = append(without, d)
			}
		}
		if original == nil {
			t.Fatal("missing original export")
		}
		if a.validateInternalModels(without) == nil {
			t.Fatal("missing declaration authorized", name)
		}
		var replacement Declaration
		if name == c.Name {
			replacement = &ValueAlias{ClassName: name, Type: c.AliasType}
		} else {
			replacement = &ValueAlias{ClassName: name, Type: TypeRef{Kind: Scalar, Name: "kotlin.time.Instant"}.Null()}
		}
		if a.validateInternalModels(append(append([]Declaration{}, without...), replacement)) == nil {
			t.Fatal("same-name copied declaration authorized", name)
		}
		if name == stamp.Name {
			// Temporal authority counts actual retained declaration identities; a
			// repeated slice pointer still denotes the same declaration.
			a.temporalOrigins[replacement] = a.temporalOrigins[original]
			if a.validateInternalModels(append(append([]Declaration{}, ds...), replacement)) == nil {
				t.Fatal("duplicate temporal identity authorized")
			}
			delete(a.temporalOrigins, replacement)
		} else if a.validateInternalModels(append(append([]Declaration{}, ds...), original)) == nil {
			t.Fatal("duplicate original declaration authorized", name)
		}
	}
	for _, owner := range []*component{c, stamp} {
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
	saved := *backing.Collection.Minimum
	*backing.Collection.Minimum++
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed original check authorized")
	}
	*backing.Collection.Minimum = saved
	originalSerializer := node.element.element.ty.Serializer
	node.element.element.ty.Serializer = "provided.Custom" // local plan cannot replace resolver authority
	if _, err := a.referencedDefaultSchemaRole(stamp.Schema, stamp.Pointer, node.element.element.ty, 0, false, referencedCollectionOwnDefault); err == nil {
		t.Fatal("custom codec authorized")
	}
	node.element.element.ty.Serializer = originalSerializer
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("immutable source tree changed")
	}
}
