package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

const internalCollectionMemberInput = `{"openapi":"3.1.0","components":{"schemas":{"Target":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]},"Owner":{"properties":{"rows":{"type":"array","nullable":true,"items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"}}}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/rows","nullable":false},"Holder":{"properties":{"v":{"properties":{"direct":{"$ref":"#/components/schemas/Owner/properties/rows","nullable":false},"alias":{"$ref":"#/components/schemas/Alias"}},"default":{"direct":[{"k":{"a":"x"}}],"alias":[{"k":{"a":"y"}}]}}}}}}}`

func TestInternalCollectionMemberSourceAndBudgets(t *testing.T) {
	a, root := referencedCollectionTestAnalyzer(t, []byte(internalCollectionMemberInput))
	properties := get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties")
	owner := a.components["Owner"]
	rows := get(get(owner.Schema, "properties"), "rows")
	target := get(referencedSchemas(root), "Target")
	before, _ := root.MarshalJSON()
	collections, nominals, uses := len(a.borrowedCollections), len(a.deferredNominals), len(a.aliasUses)
	directSteps := 0
	for _, name := range []string{"direct", "alias"} {
		member := get(properties, name)
		work := 0
		a.objectDefaultWork = &work
		node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
		if err != nil {
			t.Fatal(err)
		}
		if node.schema != rows || node.pointer != "/components/schemas/Owner/properties/rows" || node.ty.Kind == Nullable || node.element.element.schema != target || node.element.element.objectFields[0].node.schema != get(get(get(target, "allOf").Items()[0], "properties"), "a") {
			t.Fatal("lost original source/nullable", node)
		}
		extra := 0
		if name == "direct" {
			directSteps = work
		} else {
			extra = 1
			if work != directSteps+1 {
				t.Fatal("alias not charged", directSteps, work)
			}
		}
		steps := work
		for _, spare := range []int{steps, steps - 1} {
			work = maxNestedDefaultNodes - spare
			_, err = a.objectDefaultPropertyNode(member, "/member", 1, true)
			if spare == steps {
				if err != nil || work != maxNestedDefaultNodes {
					t.Fatal(work, err)
				}
			} else if err == nil || err.Code != "E_MODEL_LIMIT" || err.Pointer == nil || *err.Pointer != "/components/schemas/Target/allOf/1/properties/b" {
				t.Fatal("shared budget reset", err)
			}
		}
		a.objectDefaultWork = nil
		if _, err = a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth-6-extra, true); err != nil {
			t.Fatal("exact depth rejected", err)
		}
		if _, err = a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth-5-extra, true); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("depth lost", err)
		}
		a.hierarchySources[owner.Schema] = true
		if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil {
			t.Fatal("hierarchy source admitted")
		}
		delete(a.hierarchySources, owner.Schema)
		if _, err = a.objectDefaultChild(member, "/items", 1); err == nil {
			t.Fatal("member role leaked")
		}
	}
	if a.objectDefaultAllOfMemberElement != nil || collections != len(a.borrowedCollections) || nominals != len(a.deferredNominals) || uses != len(a.aliasUses) {
		t.Fatal("permission/binding leaked")
	}
	encoded, _ := rows.MarshalJSON()
	clone, err := document.Load(encoded, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	pointer := "/components/schemas/Owner/properties/rows"
	if err := a.internalDefaultMemberSource(clone, pointer, "#"+pointer); err == nil {
		t.Fatal("copied node admitted")
	}
	if err := a.internalDefaultMemberSource(rows, pointer, "#/components/schemas/Owner/properties/other"); err == nil {
		t.Fatal("wrong pointer admitted")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote sources")
	}
}

func TestInternalCollectionMemberRetainedDeclarations(t *testing.T) {
	root, err := document.Load([]byte(internalCollectionMemberInput), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	pending := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"TargetDto", "AliasDto"} {
		var retained Declaration
		var missing []Declaration
		for _, d := range ds {
			if d.Name() == name {
				retained = d
			} else {
				missing = append(missing, d)
			}
		}
		if retained == nil {
			t.Fatal("missing baseline", name)
		}
		check := a.validateBorrowedAllOfLeaves
		if name == "AliasDto" {
			check = a.validateBorrowedCollectionBindings
		}
		if err := check(missing); err == nil {
			t.Fatal("missing retained declaration admitted", name)
		}
		if err := check(append(append([]Declaration{}, ds...), retained)); err == nil {
			t.Fatal("duplicate retained declaration admitted", name)
		}
		if err := check(ds); err != nil {
			t.Fatal(err)
		}
	}
}
