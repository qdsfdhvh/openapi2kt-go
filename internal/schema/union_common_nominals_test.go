package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestUnionCommonNominalActualOwnershipAndNoSourceMutation(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"value":{"$ref":"#/components/schemas/Value"}},"required":["value"]},{"properties":{"kind":{"const":"b"},"value":{"$ref":"#/components/schemas/Value"}},"required":["value"]}]},"Value":{"oneOf":[{"properties":{"kind":{"const":"id"},"id":{"type":"string"}},"required":["id"]},{"properties":{"kind":{"const":"flag"},"flag":{"type":"boolean"}},"required":["flag"]}]}}}}`)
	root, e := document.Load(input, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	before, _ := root.MarshalJSON()
	a := &analyzer{root: root, options: Options{PackageName: "org.example.commonnominal", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	original := get(get(get(root, "components"), "schemas"), "Value")
	if len(a.components) != 2 || a.components["Value"].Schema != original {
		t.Fatal("fabricated catalog")
	}
	choice := ds[0].(*ObjectUnion)
	common := commonProperty(choice.Common, "value")
	if common == nil || common.Type.Name != "org.example.commonnominal.models.ValueDto" || common.Type.deferredNominal != nil {
		t.Fatal(common)
	}
	if e := a.validateInternalModels(ds); e != nil {
		t.Fatal(e)
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d.Name() != "ValueDto" {
			missing = append(missing, d)
		}
	}
	if e := a.validateInternalModels(missing); e == nil || e.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("missing actual common union accepted", e)
	}
	// Same spelling on a replacement declaration cannot satisfy original source.
	replacement := &ObjectUnion{ClassName: "ValueDto"}
	if e := a.linkNominalDeclarations(map[string]*Hierarchy{}, append(missing, replacement), true); e == nil || e.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("replacement actual common owner accepted", e)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source changed")
	}
}
