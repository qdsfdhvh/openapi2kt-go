package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestFalseDTOOwnerRootActualEdgeAndFactoryAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{"required":["rows"],"minProperties":1,"maxProperties":2,"properties":{"blocked":false,"value":{"$ref":"#/components/schemas/Maybe"},"rows":{"type":"array","items":false,"default":[]},"mapping":{"additionalProperties":{"additionalProperties":{"type":"array","nullable":true,"items":false}}}},"default":{"rows":[]}},"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Target"}},"default":{"target":{"rows":[],"mapping":{"description":{"items":null}}}}}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.ownerroot", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var owner *Model
	for model := range a.falseDTOOwnerDefaults {
		if model.ClassName == "OuterDto" {
			owner = model
		}
	}
	if owner == nil {
		t.Fatal("missing actual owner")
	}
	origin := a.falseDTOOwnerDefaults[owner]
	edge := origin.Edges[0]
	if origin.Factory == nil || edge == nil || !edge.Used || !edge.CountsUsed || edge.Owner != owner || edge.Property != &owner.Properties[0] || edge.TargetComponent != a.components["Target"] {
		t.Fatal("missing actual root/member/target/count/factory proof")
	}
	check := func(label string, change, restore func()) {
		t.Helper()
		change()
		if a.validateFalseDTOOwnerDefaults(ds) == nil {
			t.Error("changed provenance authorized", label)
		}
		restore()
		if err := a.validateFalseDTOOwnerDefaults(ds); err != nil {
			t.Fatal("restored original rejected", label, err)
		}
	}
	check("missing last owner", func() { delete(a.falseDTOOwnerDefaults, owner) }, func() { a.falseDTOOwnerDefaults[owner] = origin })
	check("missing member", func() { delete(origin.Edges, 0) }, func() { origin.Edges[0] = edge })
	factory := origin.Factory
	check("missing factory", func() { origin.Factory = nil }, func() { origin.Factory = factory })
	nodes := origin.Nodes
	check("missing owner source", func() { origin.Nodes = nil }, func() { origin.Nodes = nodes })
	literalNodes := factory.Nodes
	check("missing complete literal", func() { factory.Nodes = nil }, func() { factory.Nodes = literalNodes })
	encoded := owner.ComponentDefault
	check("changed canonical factory", func() { owner.ComponentDefault = "{}" }, func() { owner.ComponentDefault = encoded })
	check("missing leaf walk", func() { edge.Used = false }, func() { edge.Used = true })
	check("missing count walk", func() { edge.CountsUsed = false }, func() { edge.CountsUsed = true })
	count := edge.Count
	copyCount := *count
	check("substituted target counter", func() { edge.Count = &copyCount }, func() { edge.Count = count })
	targetComponent := edge.TargetComponent
	copyComponent := *targetComponent
	check("same-name target substitution", func() { edge.TargetComponent = &copyComponent }, func() { edge.TargetComponent = targetComponent })
	property := edge.Property
	copyProperty := *property
	check("copied getter", func() { edge.Property = &copyProperty }, func() { edge.Property = property })
	target := edge.Target
	copyTarget := *target
	check("copied target", func() { edge.Target = &copyTarget }, func() { edge.Target = target })
	refFactory := a.falseObjectFactories[target]
	check("missing independent reference factory", func() { delete(a.falseObjectFactories, target) }, func() { a.falseObjectFactories[target] = refFactory })
	literalFactory := a.literalFalseObjectFactories[target]
	check("missing independent literal factory", func() { delete(a.literalFalseObjectFactories, target) }, func() { a.literalFalseObjectFactories[target] = literalFactory })
	copyRoot, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	copyOwner := get(get(get(copyRoot, "components"), "schemas"), "Outer")
	copyField := get(get(copyOwner, "properties"), "target")
	properties := get(origin.Schema, "properties")
	for i, member := range properties.Members() {
		if member.Name == "target" {
			check("same-text original ref edge", func() { properties.Members()[i].Value = copyField }, func() { properties.Members()[i] = member })
		}
	}
	data := get(get(get(get(origin.Schema, "default"), "target"), "mapping"), "description")
	copiedData := get(get(get(get(copyOwner, "default"), "target"), "mapping"), "description")
	for i, member := range data.Members() {
		check("same-text metadata-named literal null", func() { data.Members()[i].Value = get(copiedData, member.Name) }, func() { data.Members()[i] = member })
	}
	if a.falseDTOOwnerDefaultScope != nil || a.componentDefaultCountRoot != nil {
		t.Fatal("root-member permission escaped its validation copy")
	}
}
