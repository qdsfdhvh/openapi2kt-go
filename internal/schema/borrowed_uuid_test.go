package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestBorrowedUUIDOriginalSourceAndExport(t *testing.T) {
	root := loadComponentDocument(t, `{"Source":{"properties":{"id":{"type":["string","null"],"format":"uuid","minLength":2},"hop":{"$ref":"#/components/schemas/Source/properties/id","nullable":false}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Bridge"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Bridge"},"uniqueItems":true}}}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.uuid", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	bridge, source := a.components["Bridge"], a.components["Source"]
	origin := bridge.InternalModelTarget.BorrowedScalar
	leaf := get(get(source.Schema, "properties"), "id")
	if origin == nil || origin.Schema != leaf || origin.Owner != source || origin.Pointer != "/components/schemas/Source/properties/id" || bare(origin.Type).Name != "kotlin.uuid.Uuid" || origin.Checks == nil || a.validationSources[origin.Checks] != origin.Pointer || len(a.components) != 3 {
		t.Fatal("borrowed UUID lost original schema/owner/check authority", origin)
	}
	if a.componentType(bridge).Kind != Scalar {
		t.Fatal("UUID internal false intent was lost")
	}
	var exported Declaration
	var missing []Declaration
	for _, d := range ds {
		if d.Name() == bridge.Name {
			exported = d
		} else {
			missing = append(missing, d)
		}
	}
	if exported == nil || a.validateBorrowedRootBindings(missing) == nil {
		t.Fatal("missing actual UUID export authorized its source")
	}
	replacement := &ValueAlias{ClassName: bridge.Name, Type: a.componentType(bridge)}
	if a.validateBorrowedRootBindings(append(missing, replacement)) == nil {
		t.Fatal("coincident UUID export name/type replaced actual declaration")
	}
	copy, err := document.Load([]byte(`{"properties":{"id":{"type":["string","null"],"format":"uuid","minLength":2},"hop":{"$ref":"#/components/schemas/Source/properties/id","nullable":false}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	original := source.Schema
	source.Schema = copy
	if a.validateBorrowedRootBindings(ds) == nil {
		t.Fatal("same-byte source clone authorized borrowed UUID")
	}
	source.Schema = original
	if err := a.validateBorrowedRootBindings(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("borrowed UUID mutated input tree")
	}
}
