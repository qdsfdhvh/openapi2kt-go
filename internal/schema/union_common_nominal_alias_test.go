package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestCommonNominalOriginalAliasAndDTOCannotBeReplaced(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/Value"},"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"value":{"$ref":"#/components/schemas/Alias"}},"required":["value"]},{"properties":{"kind":{"const":"b"},"value":{"$ref":"#/components/schemas/Alias"}},"required":["value"]}]},"Value":{"properties":{"id":{"type":"string"}},"required":["id"]}}}}`)
	root, e := document.Load(input, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.commonnominal", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	if e := a.validateInternalModels(ds); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"AliasDto", "ValueDto"} {
		missing := []Declaration{}
		var original Declaration
		for _, d := range ds {
			if d.Name() == name {
				original = d
			} else {
				missing = append(missing, d)
			}
		}
		if e := a.validateInternalModels(missing); e == nil || e.Code != "E_REF_UNSUPPORTED" {
			t.Fatal("missing common-field actual source declaration", name, e)
		}
		var replacement Declaration
		switch value := original.(type) {
		case *Model:
			copy := *value
			replacement = &copy
		case *ValueAlias:
			copy := *value
			replacement = &copy
		default:
			t.Fatal(original)
		}
		if e := a.validateInternalModels(append(missing, replacement)); e == nil || e.Code != "E_REF_UNSUPPORTED" {
			t.Fatal("same-text replacement authorized", name, e)
		}
		if e := a.validateInternalModels(append(ds, original)); e == nil || e.Code != "E_REF_UNSUPPORTED" {
			t.Fatal("duplicate actual declaration authorized", name, e)
		}
	}
}
