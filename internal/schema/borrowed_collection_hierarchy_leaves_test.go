package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestBorrowedCollectionHierarchyLeavesActualLineageAndSource(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind","mapping":{"leaf":"#/components/schemas/Leaf"}}},"Leaf":{"properties":{"id":{"type":"string"}},"required":["id"]},"Maybe":{"$ref":"#/components/schemas/Root","nullable":true},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Maybe"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/Leaf","nullable":true}}}},"RowsBridge":{"$ref":"#/components/schemas/Source/properties/rows"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.hierarchyleaves", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.borrowedHierarchyLeaves) != 2 {
		t.Fatal("hierarchy sources not retained", len(a.borrowedHierarchyLeaves))
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var parent, leaf *Hierarchy
	for _, d := range ds {
		if h, ok := d.(*Hierarchy); ok {
			if h.ClassName == "RootDto" {
				parent = h
			}
			if h.ClassName == "LeafDto" {
				leaf = h
			}
		}
	}
	if parent == nil || leaf == nil {
		t.Fatal("actual hierarchy missing")
	}
	without := []Declaration{}
	for _, d := range ds {
		if d != parent {
			without = append(without, d)
		}
	}
	for _, bad := range [][]Declaration{without, append(append([]Declaration{}, without...), &Hierarchy{ClassName: parent.ClassName}), append(append([]Declaration{}, ds...), parent)} {
		if a.validateInternalModels(bad) == nil {
			t.Fatal("missing/replaced/repeated actual hierarchy authorized")
		}
	}
	check := func(message string, change func(), restore func()) {
		t.Helper()
		change()
		if a.validateInternalModels(ds) == nil {
			t.Fatal(message)
		}
		restore()
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal("restored original lineage rejected", message, err)
		}
	}
	oldParent := leaf.Parent
	check("changed physical inheritance authorized", func() { leaf.Parent = "org.example.hierarchyleaves.models.Missing" }, func() { leaf.Parent = oldParent })
	oldField := parent.TagField
	check("changed parent dispatch tag authorized", func() { parent.TagField = "different" }, func() { parent.TagField = oldField })
	oldTags := parent.Tags
	check("changed parent known tag edges authorized", func() { parent.Tags = []HierarchyTag{{Value: "different", Child: "LeafDto"}} }, func() { parent.Tags = oldTags })
	oldLeaf := leaf.Leaf
	check("same-shaped leaf model replacement authorized", func() { copy := *oldLeaf; leaf.Leaf = &copy }, func() { leaf.Leaf = oldLeaf })
	oldEnum := parent.Enums[0]
	check("same-shaped enum declaration replacement authorized", func() { copy := *oldEnum; parent.Enums[0] = &copy }, func() { parent.Enums[0] = oldEnum })
	c := a.components["Root"]
	oldSchema := c.Schema
	bytes, _ := oldSchema.MarshalJSON()
	copied, e := document.Load(bytes, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	check("same-byte catalog schema replacement authorized", func() { c.Schema = copied }, func() { c.Schema = oldSchema })
	branches := get(oldSchema, "oneOf")
	oldBranch := branches.Items()[0]
	bytes, _ = oldBranch.MarshalJSON()
	copied, e = document.Load(bytes, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	check("same-byte branch identity replacement authorized", func() { branches.Items()[0] = copied }, func() { branches.Items()[0] = oldBranch })
	alias := a.components["Maybe"]
	height := alias.AliasHeight
	check("changed hierarchy alias source-height authorized", func() { alias.AliasHeight++ }, func() { alias.AliasHeight = height })
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original source changed")
	}
}
