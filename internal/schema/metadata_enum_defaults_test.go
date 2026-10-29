package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestMetadataEnumFactoryAndOriginalDeclaration(t *testing.T) {
	root := loadComponentDocument(t, `{"Value":{"type":"string","format":"date","enum":["ready","future"],"default":"ready"},"Alias":{"$ref":"#/components/schemas/Value","nullable":true,"default":null},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}],"default":[null,"ready"]},"Holder":{"properties":{"state":{"type":"string","format":"byte","enum":["ready"]}}},"Borrowed":{"$ref":"#/components/schemas/Holder/properties/state"}}`)
	a := analyzer{root: root, options: Options{PackageName: "org.example.metadata", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Models: ds}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	var actual *StringEnum
	for _, d := range ds {
		if d.Name() == "ValueDto" {
			actual = d.(*StringEnum)
		}
	}
	if actual == nil || actual.ComponentDefault != `"ready"` {
		t.Fatal("missing real lexical root factory")
	}
	actual.ComponentDefault = `"future"`
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("changed factory retained tuple declaration authority")
	}
	actual.ComponentDefault = `"ready"`
	if a.recordMetadataEnumFactory(actual, a.components["Value"], `"forged"`) == nil {
		t.Fatal("factory snapshot accepted a different source literal")
	}
	member := actual.Members[0]
	actual.Members[0].Wire = "changed"
	if a.recordMetadataEnumFactory(actual, a.components["Value"], `"ready"`) == nil {
		t.Fatal("factory attachment overwrote the captured vocabulary")
	}
	actual.Members[0] = member
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	// A same-byte literal copy supplies no original annotation authority.
	v := a.components["Value"].Schema
	var index int
	for i, m := range v.Members() {
		if m.Name == "default" {
			index = i
		}
	}
	original := v.Members()[index]
	copy, loadErr := document.Load([]byte(`{"value":"ready"}`), document.JSON, "")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	v.Members()[index].Value = get(copy, "value")
	if a.validateInternalModels(ds) == nil {
		t.Fatal("copied default node acquired source permission")
	}
	v.Members()[index] = original
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}

func TestMetadataEnumNativeNamedUniqueSource(t *testing.T) {
	root := loadComponentDocument(t, `{"Value":{"type":"string","enum":["ready","future"],"x-native-type":"date"},"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Value"},"uniqueItems":true,"default":["ready","future"]}}}}`)
	_, _, err := analyzeDocument(root, Options{PackageName: "org.example.metadata", Suffix: "Dto", Validation: true}, false, nil)
	if err != nil {
		t.Fatal(err)
	}
}
