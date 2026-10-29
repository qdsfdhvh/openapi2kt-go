package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestBoundedEnumObjectDefaultOriginalLeafPermission(t *testing.T) {
	root, failure := document.Load([]byte(`{"enum":["one","x"],"format":"date","minLength":2}`), document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	a := objectDefaultTestAnalyzer(true)
	node, err := a.nestedDefaultSchemaWithEnumBounds(root, "Validation", "/source/state", nil, nil, 1, true, root)
	if err != nil || node.schema != root || node.ty.Validation == nil || node.ty.Validation.MinLength == nil {
		t.Fatal("lost original bounded enum node/check", node, err)
	}
	value, loadError := document.Load([]byte(`{"value":"x"}`), document.JSON, "")
	if loadError != nil {
		t.Fatal(loadError)
	}
	if _, err := node.value(get(value, "value"), "/outer/default/a~1~0"); err == nil || err.Code != "E_SCHEMA_INVALID" || err.Pointer == nil || *err.Pointer != "/outer/default/a~1~0" {
		t.Fatal("bounded lexical annotation lost consuming Pointer", err)
	}
	copy, _ := document.Load([]byte(`{"enum":["one","x"],"format":"date","minLength":2}`), document.JSON, "")
	if _, err := a.nestedDefaultSchemaWithEnumBounds(root, "Validation", "/source/state", nil, nil, 1, true, copy); err == nil {
		t.Fatal("same-byte copied leaf acquired original bounds permission")
	}
	if _, err := a.nestedDefaultSchema(root, "Validation", "/source/state", nil, nil, 1, true); err == nil {
		t.Fatal("unscoped nested leaf acquired direct object-property bounds")
	}
}

func TestObjectDefaultOriginalFactoryMarker(t *testing.T) {
	root := loadComponentDocument(t, `{"Value":{"properties":{"state":{"enum":["one","x"],"minLength":2}},"default":{"state":"one"}}}`)
	a := analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	if _, err := a.analyze(root); err != nil {
		t.Fatal(err)
	}
	v := a.components["Value"].Schema
	if _, err := a.objectDefaultNode(v, "/components/schemas/Value", 1, false); err != nil {
		t.Fatal("original independently annotated target rejected", err)
	}
	delete(a.componentDefaultSchemas, v)
	if _, err := a.objectDefaultNode(v, "/components/schemas/Value", 1, false); err == nil {
		t.Fatal("unmarked target acquired component default ownership")
	}
	a.componentDefaultSchemas[v] = true
	copy, _ := document.Load([]byte(`{"properties":{"state":{"enum":["one","x"],"minLength":2}},"default":{"state":"one"}}`), document.JSON, "")
	a.componentDefaultSchemas[copy] = true
	if _, err := a.objectDefaultNode(copy, "/components/schemas/Value", 1, false); err == nil {
		t.Fatal("copied component marker acquired original source authority")
	}
}
