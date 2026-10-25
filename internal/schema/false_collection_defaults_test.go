package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestFalseCollectionDefaultFactoryAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Blocked":false,"Nullable":{"$ref":"#/components/schemas/Blocked","nullable":true},"Rows":{"additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/Nullable"}},"default":{"description":[null]}}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.falsedefaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var alias *ValueAlias
	for _, d := range ds {
		if value, ok := d.(*ValueAlias); ok && value.ClassName == "RowsDto" {
			alias = value
		}
	}
	if alias == nil || alias.ComponentDefault != `{"description":[null]}` {
		t.Fatal("missing original canonical factory", alias)
	}
	encoded := alias.ComponentDefault
	alias.ComponentDefault = `{}`
	if a.validateInternalModels(ds) == nil {
		t.Error("changed actual factory initializer authorized")
	}
	alias.ComponentDefault = encoded
	proof := a.falseCollectionFactories[alias]
	delete(a.falseCollectionFactories, alias)
	if a.validateInternalModels(ds) == nil {
		t.Error("missing actual factory authority authorized")
	}
	a.falseCollectionFactories[alias] = proof
	copyRoot, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	copied := get(get(get(copyRoot, "components"), "schemas"), "Rows")
	array := get(get(a.components["Rows"].Schema, "default"), "description")
	copyArray := get(get(copied, "default"), "description")
	saved := array.Items()[0]
	array.Items()[0] = copyArray.Items()[0]
	if a.validateInternalModels(ds) == nil {
		t.Error("same-byte default literal hidden by schema metadata authorized")
	}
	array.Items()[0] = saved
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
