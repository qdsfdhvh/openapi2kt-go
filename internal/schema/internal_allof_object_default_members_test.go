package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestInternalAllOfObjectDefaultMemberAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"node":{"allOf":[{"properties":{"label":{"type":"string"}},"required":["label"]},{"properties":{"n":{"type":"integer"}}}]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Other":{"properties":{"x":{"type":"string"}}},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Alias"}},"default":{"member":{"label":"x"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	field := get(get(get(referencedSchemas(root), "Holder"), "properties"), "v")
	owner := a.components["Owner"]
	target := get(get(owner.Schema, "properties"), "node")
	binding := a.deferredNominals[target]
	if binding == nil || binding.Name != "org.example.defaults.models.OwnerDto.Node" {
		t.Fatal(binding)
	}
	before, _ := root.MarshalJSON()
	work := 0
	a.objectDefaultWork = &work
	node, err := a.objectDefaultNode(field, "/field", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	child := node.objectFields[0].node
	if child.schema != target || child.pointer != "/components/schemas/Owner/properties/node" || child.objectFields[0].node.schema != get(get(get(target, "allOf").Items()[0], "properties"), "label") {
		t.Fatal("lost original source", child)
	}
	steps := work
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err = a.objectDefaultNode(field, "/field", 1, true)
		if spare == steps {
			if err != nil || work != maxNestedDefaultNodes {
				t.Fatal(work, err)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" || err.Pointer == nil || *err.Pointer != "/components/schemas/Owner/properties/node/allOf/1/properties/n" {
			t.Fatal("reset internal source budget", err)
		}
	}
	a.objectDefaultWork = nil
	if _, err = a.objectDefaultNode(field, "/field", MaxSchemaDepth-3, true); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("lost incoming/alias depth", err)
	}
	ref := get(get(field, "properties"), "member")
	original := *binding
	for _, mutate := range []func(){
		func() { binding.Owner = a.components["Other"] },
		func() { binding.Pointer = "/components/schemas/Other/properties/x" },
		func() { binding.Schema = owner.Schema },
		func() { binding.Name = "" },
		func() { a.hierarchySources[owner.Schema] = true },
		func() { delete(a.deferredNominals, target) },
	} {
		mutate()
		if _, err = a.objectDefaultPropertyReference(ref, "/member", 1); err == nil {
			t.Fatal("invalid source authority admitted")
		}
		*binding = original
		a.deferredNominals[target] = binding
		delete(a.hierarchySources, owner.Schema)
	}
	if _, err = a.objectDefaultChild(ref, "/items", 1); err == nil {
		t.Fatal("member permission leaked into container")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original source")
	}
}

func TestInternalAllOfObjectDefaultMemberFinalDeclaration(t *testing.T) {
	for _, collection := range []bool{false, true} {
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Owner/properties/node"}},"default":{"member":{"label":"x"}}}}},"Owner":{"properties":{"node":{"allOf":[{"properties":{"label":{"type":"string"}},"required":["label"]},{"properties":{"n":{"type":"integer"}}}]}}}}}}`)
		if collection {
			input = []byte(strings.ReplaceAll(string(input), `"member":{"$ref":"#/components/schemas/Owner/properties/node"}`, `"member":{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/node"}}`))
			input = []byte(strings.ReplaceAll(string(input), `"member":{"label":"x"}`, `"member":[{"label":"x"}]`))
		}
		root, err := document.Load(input, document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		pending := []referencedFieldDefault{}
		a := &analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
		ds, err := a.analyze(root)
		if err != nil {
			t.Fatal(err)
		}
		// Holder sorts before Owner: its annotation is checked provisionally, but
		// emission still requires the original unique retained Owner.Node Model.
		var owner Declaration
		var missing []Declaration
		for _, d := range ds {
			if d.Name() == "OwnerDto" {
				owner = d
			} else {
				missing = append(missing, d)
			}
		}
		if owner == nil {
			t.Fatal("missing owner")
		}
		if err := a.linkNominalDeclarations(map[string]*Hierarchy{}, missing, true); err == nil || err.Code != "E_REF_UNSUPPORTED" {
			t.Fatal("missing retained owner authorized", err)
		}
		repeated := append(append([]Declaration{}, ds...), owner)
		if err := a.linkNominalDeclarations(map[string]*Hierarchy{}, repeated, true); err == nil || err.Code != "E_REF_UNSUPPORTED" {
			t.Fatal("ambiguous owner authorized", err)
		}
		if err := a.linkNominalDeclarations(map[string]*Hierarchy{}, ds, true); err != nil {
			t.Fatal(err)
		}
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal(err)
		}
	}
}
