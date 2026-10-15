package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestNamedCollectionReferenceChainsOriginalBackingIdentityAndIncomingHeight(t *testing.T) {
	for _, terminal := range []string{`{"type":["array","null"],"minItems":1,"items":{"type":"string","minLength":1}}`, `{"nullable":true,"additionalProperties":{"type":"string","minLength":1},"minProperties":1}`, `{"nullable":true,"additionalProperties":true}`, `{"type":"array","items":{"properties":{"id":{"type":"string"}},"required":["id"]}}`} {
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":` + terminal + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Alias"},"false":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false},"again":{"$ref":"#/components/schemas/Bridge","nullable":true}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/false","nullable":true},"Again":{"$ref":"#/components/schemas/Hops/properties/again","nullable":false}}}}`)
		root, err := document.Load(input, document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		before, _ := root.MarshalJSON()
		a := analyzer{root: root, options: Options{PackageName: "org.example.collectionchains", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
		ds, err := a.analyze(root)
		if err != nil {
			t.Fatal(terminal, err)
		}
		c := a.components["Value"]
		for _, use := range []struct {
			path     string
			steps    int
			nullable bool
		}{{"Hops/properties/value", 3, true}, {"Hops/properties/false", 4, false}, {"Hops/properties/again", 6, true}, {"Again", 7, false}} {
			target, err := a.resolveSchema("#/components/schemas/"+use.path, "/use")
			if err != nil {
				t.Fatal(err)
			}
			if target.CanonicalCollection != c || target.Schema != c.Schema || target.Pointer != c.Pointer || target.Steps != use.steps || target.ModelName != "" || target.Deferred != nil || target.CanonicalJSON != nil || target.BorrowedScalar != nil {
				t.Fatal("namedcontainer disguised as borrowedscalar/nominal", use, target)
			}
			if flag, err := target.effectiveNullable(); err != nil || flag != use.nullable {
				t.Fatal(use, flag, err)
			}
			a.aliasPlanning = true
			no := false
			ty, err := a.referenceType(target, &no, 48-use.steps-c.CollectionHeight, "/48")
			a.aliasPlanning = false
			if err != nil || ty.Kind != Named || ty.Element != &c.AliasType || ty.ArrayAliasView != NonNullArrayAlias || ty.Slots() != 1 {
				t.Fatal("copiedunlinkedcontainerbacking", use, ty, err)
			}
			if err := a.aliasConsumerDepth(); err != nil {
				t.Fatal("fullsource48 rejected", terminal, use, err)
			}
			if _, err := a.referenceType(target, &no, 49-use.steps-c.CollectionHeight, "/49"); err != nil {
				continue
			}
			if err := a.aliasConsumerDepth(); err == nil {
				t.Fatal("cachedcontainerfullsource49 admitted", use, c.CollectionHeight)
			}
			delete(a.aliasUses, collectionAliasUse{c, 49 - use.steps - c.CollectionHeight + use.steps - 1, "/49"})
		}
		missing := []Declaration{}
		var original Declaration
		for _, d := range ds {
			if d.Name() == c.Name {
				original = d
			} else {
				missing = append(missing, d)
			}
		}
		for peer, owner := range a.declarationOwners {
			if owner != c {
				continue
			}
			withoutPeer := []Declaration{}
			for _, d := range ds {
				if d != peer {
					withoutPeer = append(withoutPeer, d)
				}
			}
			if err := a.validateNamedCollectionBindings(withoutPeer); err == nil {
				t.Fatal("missing original item peer still authorized")
			}
			if err := a.validateNamedCollectionBindings(append(append([]Declaration{}, ds...), peer)); err == nil {
				t.Fatal("duplicate original item peer still authorized")
			}
		}
		if err := a.validateNamedCollectionBindings(missing); err == nil {
			t.Fatal("missing actualnamedcollectionalias authorized")
		}
		copy := &ValueAlias{ClassName: c.Name, Type: c.AliasType}
		if err := a.validateNamedCollectionBindings(append(append([]Declaration{}, missing...), copy)); err == nil {
			t.Fatal("samenamecollectionaliasreplacement authorized")
		}
		if err := a.validateNamedCollectionBindings(append(append([]Declaration{}, ds...), original)); err == nil {
			t.Fatal("sameoriginalalias duplicate occurrence authorized")
		}
		saved := c.Schema
		copyroot, err := document.Load([]byte(`{"value":`+terminal+`}`), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		c.Schema = get(copyroot, "value")
		if err := a.validateNamedCollectionBindings(ds); err == nil {
			t.Fatal("samebytecollectionsource replacement authorized")
		}
		c.Schema = saved
		if backing := c.AliasType.WithoutRootNull(); backing.Kind == List || backing.Kind == Map {
			child := backing.Element
			savedName := child.Name
			child.Name = "kotlin.Boolean"
			if err := a.validateNamedCollectionBindings(ds); err == nil {
				t.Fatal("mutatedshareditembacking authorized")
			}
			child.Name = savedName
		}
		for _, source := range a.namedCollectionOrigins[original].Types {
			if source.Node.Validation != nil && source.Node.Validation.MinLength != nil {
				value := source.Node.Validation.MinLength
				saved := *value
				*value++
				if err := a.validateNamedCollectionBindings(ds); err == nil {
					t.Fatal("mutated original item checks authorized")
				}
				*value = saved
			}
			if source.Node.Collection != nil && source.Node.Collection.Minimum != nil {
				value := source.Node.Collection.Minimum
				saved := *value
				*value++
				if err := a.validateNamedCollectionBindings(ds); err == nil {
					t.Fatal("mutated original container checks authorized")
				}
				*value = saved
			}
		}
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal(err)
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("originalcollectionschema mutated")
		}
	}
}
