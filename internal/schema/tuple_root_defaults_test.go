package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestScalarTupleFactoryRetainsOriginalSourcesAndIndependentAliasValues(t *testing.T) {
	root := loadComponentDocument(t, `{"Rows":{"type":"array","prefixItems":[{"type":"integer"},{"type":"integer"}],"default":[1,2]},"Alias":{"$ref":"#/components/schemas/Rows","default":[3]}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.tuplefactory", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, encoded string }{{"Rows", "[1,2]"}, {"Alias", "[3]"}} {
		alias, ok := componentDeclaration(ds, tc.name+"Dto").(*ValueAlias)
		if !ok || alias.ComponentDefault != tc.encoded {
			t.Fatal("actual alias factory lost independent annotation", tc, alias)
		}
	}
	c := a.components["Rows"]
	node, err := a.componentDefaultNode(c)
	first := get(c.Schema, "prefixItems").Items()[0]
	if err != nil || node.schema != c.Schema || node.element.schema != first || node.element.pointer != c.Pointer+"/prefixItems/0" || node.ty.Kind != List {
		t.Fatal("tuple factory copied source or selected a synthetic leaf", node, err)
	}
	bytes, _ := c.Schema.MarshalJSON()
	copied, e := document.Load(bytes, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	if _, err := a.arrayItems(copied, c.Pointer, 0); err == nil {
		t.Fatal("copied root gained generic default permission")
	}
	shadow := *c
	shadow.Schema = copied
	if _, err := a.componentDefaultNode(&shadow); err == nil {
		t.Fatal("factory declaration lost original root authority")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("tuple factory rewrote source AST")
	}
}
