package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleTargetDefaultsKeepOriginalEnumAndAliasSources(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Choice":{"type":"string","enum":["ready","done"],"default":"ready"},"Alias":{"$ref":"#/components/schemas/Choice","default":"done"},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}],"items":{"type":"integer"}}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.factories", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Choice", "Alias"} {
		owner := a.components[name]
		original := owner.Schema
		bytes, _ := original.MarshalJSON()
		copied, err := document.Load(bytes, document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		owner.Schema = copied
		if a.validateInternalModels(ds) == nil {
			t.Fatal("factory permitted copied source", name)
		}
		owner.Schema = original
	}
	choice := a.components["Choice"].Schema
	for i, member := range choice.Members() {
		if member.Name != "default" {
			continue
		}
		bytes, _ := member.Value.MarshalJSON()
		wrapper, err := document.Load([]byte(`{"value":`+string(bytes)+`}`), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		choice.Members()[i].Value = get(wrapper, "value")
		if a.validateInternalModels(ds) == nil {
			t.Fatal("factory erased original default value node authority")
		}
		choice.Members()[i] = member
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("factory tuple consumed by changing original source")
	}
}
