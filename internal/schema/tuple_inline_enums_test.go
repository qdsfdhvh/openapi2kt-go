package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleInlineEnumOriginalDiscardedSource(t *testing.T) {
	for _, leaf := range []string{`{"enum":["ready","later"]}`, `{"type":"integer","enum":[1,2]}`, `{"type":"string","format":"date","enum":["ready"]}`, `{"type":"string","x-native-type":"org.example.provider.Amount","enum":["ready"]}`} {
		for _, checked := range []bool{false, true} {
			root := loadComponentDocument(t, `{"Probe":{"type":"array","prefixItems":[`+leaf+`],"items":true}}`)
			a := analyzer{root: root, options: Options{PackageName: "org.example.inlineenum", Suffix: "Dto", Validation: checked}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
			ds, err := a.analyze(root)
			if err != nil {
				t.Fatal(err)
			}
			prefix := get(a.components["Probe"].Schema, "prefixItems")
			member := prefix.Items()[0]
			copy, err := document.Load([]byte(leaf), document.JSON, "")
			if err != nil {
				t.Fatal(err)
			}
			prefix.Items()[0] = copy
			if a.validateInternalModels(ds) == nil {
				t.Fatal("same-byte member copy acquired original authority")
			}
			prefix.Items()[0] = member
			vocabulary, replacement := get(member, "enum"), get(copy, "enum")
			original := vocabulary.Items()[0]
			vocabulary.Items()[0] = replacement.Items()[0]
			if a.validateInternalModels(ds) == nil {
				t.Fatal("same-byte literal copy acquired original authority")
			}
			vocabulary.Items()[0] = original
			if err := a.validateInternalModels(ds); err != nil {
				t.Fatal(err)
			}
		}
	}
}
