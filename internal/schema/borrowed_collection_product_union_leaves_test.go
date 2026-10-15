package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestBorrowedCollectionProductUnionLeavesOriginalDeclarationsAndDispatch(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Part":{"properties":{"kind":{"const":"id"},"description":{"type":"string"}},"required":["description"]},"PartAlias":{"$ref":"#/components/schemas/Part"},"PartAgain":{"$ref":"#/components/schemas/PartAlias"},"Node":{"allOf":[{"oneOf":[{"$ref":"#/components/schemas/PartAgain"},{"properties":{"kind":{"const":"label"},"label":{"type":"string"}},"required":["label"]}]},{"properties":{"stamp":{"const":"shared","type":"string"}},"required":["stamp"]}]},"Alias":{"$ref":"#/components/schemas/Node"},"Owner":{"properties":{"choice":{"oneOf":[{"type":"string"},{"type":"boolean"},{"type":"null"}]}}},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/Owner/properties/choice"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.borrowedunion", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	if len(a.borrowedUnionLeaves) != 2 {
		t.Fatal("original union leaves missing", len(a.borrowedUnionLeaves))
	}
	for _, use := range a.borrowedUnionLeaves {
		if use.Schema == a.components["Node"].Schema {
			if !use.Product {
				t.Fatal("product union lost distinct original source role")
			}
			use.Product = false
			if a.validateInternalModels(ds) == nil {
				t.Fatal("product source authorized as a direct union source")
			}
			use.Product = true
		}
	}
	var original *ObjectUnion
	var owner *Model
	without := []Declaration{}
	for _, d := range ds {
		if d.Name() == "NodeDto" {
			original = d.(*ObjectUnion)
		} else {
			without = append(without, d)
		}
		if d.Name() == "OwnerDto" {
			owner = d.(*Model)
		}
	}
	if original == nil || owner == nil {
		t.Fatal("physical union owner missing")
	}
	for _, bad := range [][]Declaration{without, append(append([]Declaration{}, without...), &ObjectUnion{ClassName: original.ClassName, Variants: original.Variants}), append(append([]Declaration{}, ds...), original)} {
		if a.validateInternalModels(bad) == nil {
			t.Fatal("missing/replaced/repeated actual original union authorized")
		}
	}
	primitive := owner.Nested[0].(*PrimitiveUnion)
	children := owner.Nested
	owner.Nested = nil
	if a.validateInternalModels(ds) == nil {
		t.Fatal("missing nested actual primitive union authorized")
	}
	owner.Nested = append(append([]Declaration{}, children...), primitive)
	if a.validateInternalModels(ds) == nil {
		t.Fatal("repeated nested primitive union authorized")
	}
	owner.Nested = children
	origin := a.unionOrigins[primitive]
	changed := origin
	changed.Nullable = !origin.Nullable
	a.unionOrigins[primitive] = changed
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed original emitted union nullable view authorized")
	}
	a.unionOrigins[primitive] = origin
	kinds := primitive.Kinds
	primitive.Kinds = []PrimitiveKind{PrimitiveInteger}
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed canonical native union kinds authorized")
	}
	primitive.Kinds = kinds
	nullVariant := primitive.NullVariant
	primitive.NullVariant = !nullVariant
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed primitive union null carrier authorized")
	}
	primitive.NullVariant = nullVariant
	primitiveObject := primitive.Object
	primitive.Object = &Model{ClassName: "Fabricated"}
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed actual primitive union object carrier authorized")
	}
	primitive.Object = primitiveObject
	combinations := original.TagCombinations
	original.TagCombinations = []ObjectTagRule{{Index: 1}}
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed original union combined dispatch rules authorized")
	}
	original.TagCombinations = combinations
	tag := original.TagField
	original.TagField = "replacedDispatch"
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed original object union dispatch key authorized")
	}
	original.TagField = tag
	base := a.components["Part"]
	old := base.Schema
	bytes, _ := old.MarshalJSON()
	copied, e := document.Load(bytes, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	base.Schema = copied
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte original union branch target replacement authorized")
	}
	base.Schema = old
	alias := a.components["PartAlias"]
	old = alias.Schema
	bytes, _ = old.MarshalJSON()
	copied, e = document.Load(bytes, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	alias.Schema = copied
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte original union intermediate shape alias replacement authorized")
	}
	alias.Schema = old
	branches := get(get(a.components["Node"].Schema, "allOf").Items()[0], "oneOf")
	branch := branches.Items()[0]
	bytes, _ = branch.MarshalJSON()
	copied, e = document.Load(bytes, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	branches.Items()[0] = copied
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte original union branch replacement authorized")
	}
	branches.Items()[0] = branch
	for _, use := range a.borrowedUnionLeaves {
		if use.Schema == a.components["Node"].Schema {
			found := false
			prop := get(get(base.Schema, "properties"), "description")
			for _, node := range use.Nodes {
				found = found || node.Node == prop
			}
			if !found {
				t.Fatal("union branch property named description omitted from source identity")
			}
		}
	}
	leafAlias := a.components["Alias"]
	height := leafAlias.AliasHeight
	leafAlias.AliasHeight++
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed original nominal union alias suffix cache authorized")
	}
	leafAlias.AliasHeight = height
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(after) != string(before) {
		t.Fatal("borrowed union planning modified original source")
	}
}
