package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestFalseDTOOwnerAliasOriginalPathAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{"required":["rows"],"minProperties":1,"maxProperties":2,"properties":{"blocked":false,"value":{"$ref":"#/components/schemas/Maybe"},"rows":{"type":"array","items":false,"default":[]},"mapping":{"additionalProperties":{"additionalProperties":{"type":"array","nullable":true,"items":false}}}},"default":{"rows":[]}},"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Near"}},"default":{"target":{"rows":[],"mapping":{"description":{"items":null}}}}},"Far":{"$ref":"#/components/schemas/Target","nullable":true},"Near":{"$ref":"#/components/schemas/Far","nullable":false}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.aliasroot", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
		t.Fatal("missing original owner")
	}
	edge := a.falseDTOOwnerDefaults[owner].Edges[0]
	if len(edge.Aliases) != 2 || edge.Aliases[0].State.Nullable || !edge.Aliases[1].State.Nullable {
		t.Fatal("original nearest alias path lost")
	}
	check := func(label string, change, restore func()) {
		t.Helper()
		change()
		if a.validateFalseDTOOwnerDefaults(ds) == nil {
			t.Error("alias authority changed", label)
		}
		restore()
		if err := a.validateFalseDTOOwnerDefaults(ds); err != nil {
			t.Fatal("restored alias rejected", label, err)
		}
	}
	for i := range edge.Aliases {
		c := edge.Aliases[i].State.Component
		height := c.AliasHeight
		check("cached height", func() { c.AliasHeight++ }, func() { c.AliasHeight = height })
		target := c.BindingTarget
		check("missing canonical binding", func() { c.BindingTarget = nil }, func() { c.BindingTarget = target })
		nullable := *c.BindingNullable
		check("changed cached nullable", func() { *c.BindingNullable = !nullable }, func() { *c.BindingNullable = nullable })
		reference := c.Reference
		check("changed cached reference", func() { c.Reference = "#/components/schemas/Outer" }, func() { c.Reference = reference })
		nodes := edge.Aliases[i].Nodes
		check("missing alias source capture", func() { edge.Aliases[i].Nodes = nil }, func() { edge.Aliases[i].Nodes = nodes })
		saved := a.components[c.Wire]
		copyComponent := *c
		check("copied catalog entry", func() { a.components[c.Wire] = &copyComponent }, func() { a.components[c.Wire] = saved })
	}
	path := edge.Aliases
	nodes := edge.Aliases[0].Nodes
	check("another alias source capture", func() { edge.Aliases[0].Nodes = edge.Aliases[1].Nodes }, func() { edge.Aliases[0].Nodes = nodes })
	check("missing whole alias path", func() { edge.Aliases = nil }, func() { edge.Aliases = path })
	copied, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	for _, step := range edge.Aliases {
		source := step.State.Schema
		copySchema := get(get(get(copied, "components"), "schemas"), step.State.Wire)
		for i, member := range source.Members() {
			if member.Name == "$ref" || member.Name == "nullable" {
				check("same-text alias source replacement", func() { source.Members()[i].Value = get(copySchema, member.Name) }, func() { source.Members()[i] = member })
			}
		}
	}
}
