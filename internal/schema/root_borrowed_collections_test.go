package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestBorrowedCollectionRootOriginalGraphChecksAndActualAliases(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"leaf":{"type":["string","null"],"minLength":2},"hop":{"$ref":"#/components/schemas/Source/properties/leaf"},"list":{"type":["array","null"],"minItems":1,"items":{"$ref":"#/components/schemas/Source/properties/hop"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/Source/properties/list","nullable":false}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/list"},"False":{"$ref":"#/components/schemas/Bridge","nullable":false},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/False","nullable":true}}},"Again":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.borrowedcollections", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	origin := a.components["Bridge"].InternalModelTarget.BorrowedCollection
	if origin == nil || origin.Owner != a.components["Source"] || origin.Schema != get(get(origin.Owner.Schema, "properties"), "list") || len(origin.Graph) != 4 || len(a.components) != 5 {
		t.Fatal("fake collection source/catalog", origin, len(a.components))
	}
	for _, c := range []string{"False", "Again"} {
		if a.components[c].InternalModelTarget.BorrowedCollection != origin {
			t.Fatal("root source/checks not shared", c)
		}
	}
	bridge := a.components["Bridge"]
	for _, cached := range []*int{&bridge.AliasHeight, &bridge.CollectionHeight} {
		saved := *cached
		*cached++
		if a.validateBorrowedCollectionBindings(ds) == nil {
			t.Fatal("changed original borrowed graph cache still authorized")
		}
		*cached = saved
	}
	savedIntent := *bridge.BindingNullable
	*bridge.BindingNullable = !savedIntent
	if a.validateBorrowedCollectionBindings(ds) == nil {
		t.Fatal("changed borrowed root path nullable still authorized")
	}
	*bridge.BindingNullable = savedIntent
	var actual Declaration
	var missing []Declaration
	for _, d := range ds {
		if d.Name() == "BridgeDto" {
			actual = d
		} else {
			missing = append(missing, d)
		}
	}
	for _, bad := range [][]Declaration{missing, append(append([]Declaration{}, missing...), &ValueAlias{ClassName: "BridgeDto", Type: actual.(*ValueAlias).Type}), append(append([]Declaration{}, ds...), actual)} {
		if err := a.validateBorrowedCollectionBindings(bad); err == nil {
			t.Fatal("missing/replaced/repeated original root alias authorized")
		}
	}
	for _, node := range origin.Types {
		if node.Node.Validation != nil && node.Node.Validation.MinLength != nil {
			p := node.Node.Validation.MinLength
			saved := *p
			*p++
			if a.validateBorrowedCollectionBindings(ds) == nil {
				t.Fatal("mutated original borrowed leaf checks authorized")
			}
			*p = saved
		}
		if node.Node.Collection != nil && node.Node.Collection.Minimum != nil {
			p := node.Node.Collection.Minimum
			saved := *p
			*p++
			if a.validateBorrowedCollectionBindings(ds) == nil {
				t.Fatal("mutated original borrowed container checks authorized")
			}
			*p = saved
		}
	}
	backing := bare(origin.Type)
	savedName := backing.Element.Name
	backing.Element.Name = "fake"
	if a.validateBorrowedCollectionBindings(ds) == nil {
		t.Fatal("mutated linked borrowed backing authorized")
	}
	backing.Element.Name = savedName
	// Identical bytes do not authorize a replacement nested ref node. Mutate
	// the original document parent, rather than only replacing catalog fields.
	props := get(origin.Owner.Schema, "properties")
	members := props.Members()
	var index int
	for i, m := range members {
		if m.Name == "hop" {
			index = i
		}
	}
	originalHop := members[index].Value
	copied, err := document.Load([]byte(`{"value":{"$ref":"#/components/schemas/Source/properties/leaf"}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	members[index].Value = get(copied, "value")
	if a.validateBorrowedCollectionBindings(ds) == nil {
		t.Fatal("same-byte nested borrowed ref replacement authorized")
	}
	members[index].Value = originalHop
	if err := a.validateBorrowedCollectionBindings(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source mutated")
	}
}
