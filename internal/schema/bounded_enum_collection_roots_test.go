package schema

import "testing"

func TestBoundedEnumCollectionRootOriginalBackingAndPeer(t *testing.T) {
	root := loadComponentDocument(t, `{"Tree":{"type":"array","items":{"additionalProperties":{"enum":["one","two","x"],"minLength":2}},"default":[{"root":"one"}]},"Alias":{"$ref":"#/components/schemas/Tree","default":[{"alias":"two"}]},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":[{"field":"one"}]}},"default":{"value":[{"outer":"two"}]}}}`)
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.roots", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
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
	if err := a.validateObjectDefaultGraphUses(); err != nil {
		t.Fatal(err)
	}
	c := a.components["Tree"]
	leaf := get(get(c.Schema, "items"), "additionalProperties")
	var peer *StringEnum
	for _, d := range ds {
		if enum, ok := d.(*StringEnum); ok && a.enumOrigins[enum].Schema == leaf {
			peer = enum
		}
	}
	if peer == nil || a.declarationOwners[peer] != c {
		t.Fatal("root collection lost original enum peer")
	}
	terminal := c.AliasType.Element.Element
	if terminal.Validation == nil || a.validationSources[terminal.Validation] != "/components/schemas/Tree/items/additionalProperties" {
		t.Fatal("root collection lost original leaf check")
	}
	if len(pending) == 0 {
		t.Fatal("consuming annotation was not retained")
	}
	for _, use := range pending {
		node, err := a.referencedDefaultSchema(use.Field.Schema, use.Field.Pointer, use.Type, use.Field.Depth, true)
		if err != nil || node.element.element.schema != leaf || node.element.element.ty.Validation != terminal.Validation {
			t.Fatal("consuming default copied original lexical checks/source", err)
		}
	}
	original := peer.Members[0]
	peer.Members[0].Wire = "changed"
	if err := a.validateNamedCollectionBindings(ds); err == nil {
		t.Fatal("root enum peer vocabulary mutation escaped source proof")
	}
	peer.Members[0] = original
	minimum := *terminal.Validation.MinLength
	*terminal.Validation.MinLength++
	if err := a.validateNamedCollectionBindings(ds); err == nil {
		t.Fatal("mutated root leaf check escaped original source proof")
	}
	*terminal.Validation.MinLength = minimum
	for _, removed := range []Declaration{peer, componentDeclaration(ds, c.Name)} {
		var missing []Declaration
		for _, d := range ds {
			if d != removed {
				missing = append(missing, d)
			}
		}
		if err := a.validateNamedCollectionBindings(missing); err == nil {
			t.Fatal("missing root alias/enum peer escaped source proof")
		}
		if err := a.validateNamedCollectionBindings(append(append([]Declaration{}, ds...), removed)); err == nil {
			t.Fatal("duplicate root alias/enum peer escaped source proof")
		}
	}
	if err := a.validateNamedCollectionBindings(ds); err != nil {
		t.Fatal(err)
	}
}
