package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleFieldDefaultsOriginalSourcesAndScopedPermission(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"type":"integer"},{"type":"integer"}],"default":[1,2]}}}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.tupledefaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	if _, err := a.analyze(root); err != nil {
		t.Fatal(err)
	}
	field := get(get(a.components["Holder"].Schema, "properties"), "rows")
	pointer := "/components/schemas/Holder/properties/rows"
	if _, err := a.arrayItems(field, pointer, 1); err == nil || err.Pointer == nil || *err.Pointer != pointer+"/default" {
		t.Fatal("direct field default leaked generic tuple/root annotation permission", err)
	}
	decision, err := a.arrayItemsOwnDefault(field, pointer, 1, true)
	if err != nil || decision.Selected != get(field, "prefixItems").Items()[0] || decision.Pointer != pointer+"/prefixItems/0" || decision.Prefix[1].Schema != get(field, "prefixItems").Items()[1] {
		t.Fatal("tuple default replaced original positional sources", decision, err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("tuple default mutated source nodes")
	}
}
