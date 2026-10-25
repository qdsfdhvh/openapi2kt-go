package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestEnumWrapperDefaultsKeepOriginalSourceAndBoundAlias(t *testing.T) {
	root := loadComponentDocument(t, `{"State":{"enum":["on",null,"off"],"default":null},"Peer":{"enum":["on",null,"off"]},"Middle":{"$ref":"#/components/schemas/State","nullable":true,"default":null},"Narrow":{"$ref":"#/components/schemas/Middle","nullable":false,"default":"on"},"Again":{"$ref":"#/components/schemas/Narrow","nullable":true,"default":null}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	source := a.components["State"].Schema
	vocabulary := get(source, "enum")
	null := vocabulary.Items()[1]
	state, ok := componentDeclaration(ds, "StateDto").(*StringEnum)
	if !ok || state.ComponentDefault != "null" {
		t.Fatal("independent source annotation lost")
	}
	for name, expected := range map[string]string{"Middle": "null", "Narrow": `"on"`, "Again": "null"} {
		alias, ok := componentDeclaration(ds, name+"Dto").(*ValueAlias)
		if !ok || alias.ComponentDefault != expected || (alias.Type.Kind == Nullable) != (name != "Narrow") {
			t.Fatal("nearest alias intent/own factory lost", name, alias)
		}
	}
	c := a.components["Narrow"]
	target := a.referenceWrapperDefaultTarget(c)
	validation := a
	if encoded, err := a.originalEnumWrapperDefaultEncoded(c, target, &validation); err != nil || encoded != `"on"` {
		t.Fatal(encoded, err)
	}
	json, _ := source.MarshalJSON()
	copied, err := document.Load(json, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, forged := range []wrapperDefaultSource{
		{Pointer: target.Pointer, Schema: copied, Nullable: target.Nullable},
		{Pointer: "/components/schemas/Missing", Schema: source, Nullable: target.Nullable},
		{Pointer: "/components/schemas/Peer", Schema: a.components["Peer"].Schema, Nullable: target.Nullable},
		{Pointer: target.Pointer, Schema: source},
	} {
		if _, err := a.originalEnumWrapperDefaultEncoded(c, &forged, &validation); err == nil {
			t.Fatal("copied or unrelated vocabulary/intent authorized")
		}
	}
	aliasJSON, _ := c.Schema.MarshalJSON()
	aliasCopy, _ := document.Load(aliasJSON, document.JSON, "")
	shadow := *c
	shadow.Schema = aliasCopy
	if _, err := a.originalEnumWrapperDefaultEncoded(&shadow, target, &validation); err == nil {
		t.Fatal("copied consuming alias authorized")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) || get(source, "enum") != vocabulary || vocabulary.Items()[1] != null {
		t.Fatal("factory validation rewrote original vocabulary or alias")
	}
}
