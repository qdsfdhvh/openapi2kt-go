package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleInlineDTOValidationIsIsolated(t *testing.T) {
	root := loadComponentDocument(t, `{"Choice":{"enum":["ready"]},"Rows":{"type":"array","prefixItems":[{"properties":{"state":{"$ref":"#/components/schemas/Choice","default":"ready"},"nested":{"properties":{"n":{"type":"integer"}},"default":{"n":7}}}}],"items":true}}`)
	before, _ := root.MarshalJSON()
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.inlinedto", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 2 || len(pending) != 0 || len(a.modelOrigins) != 0 || len(a.enumOrigins) != 1 || len(a.components) != 2 {
		t.Fatal("discarded source leaked analysis registrations", len(ds), len(pending), len(a.modelOrigins), len(a.enumOrigins), len(a.components))
	}
	for _, d := range ds {
		if strings.Contains(d.Name(), "TupleSource") {
			t.Fatal("source validation declaration escaped")
		}
	}
	if a.tupleObjects == nil || !a.tupleObjects.Validated || a.tupleObjects.Proof == nil {
		t.Fatal("missing isolated source completion/proof")
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	proof := a.tupleObjects.Proof
	a.tupleObjects.Proof = nil
	if a.validateInternalModels(ds) == nil {
		t.Fatal("missing isolated source proof was accepted")
	}
	a.tupleObjects.Proof = proof
	a.tupleObjects.Validated = false
	if a.validateInternalModels(ds) == nil {
		t.Fatal("incomplete isolated source analysis was accepted")
	}
	a.tupleObjects.Validated = true
	member := get(a.components["Rows"].Schema, "prefixItems").Items()[0]
	props := get(member, "properties")
	original := props.Members()[0].Value
	bytes, _ := original.MarshalJSON()
	copy, failure := document.Load(bytes, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	props.Members()[0].Value = copy
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte discarded field source copy acquired proof")
	}
	props.Members()[0].Value = original
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source validation rewrote input")
	}
	plan, warnings, err := analyzeDocument(root, a.options, true, nil)
	if err != nil || plan != nil || warnings != nil {
		t.Fatal("source-only analysis returned an emission candidate", err)
	}
}
