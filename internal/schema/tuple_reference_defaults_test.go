package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleReferenceDefaultOriginalSourcesAndActualType(t *testing.T) {
	for _, target := range []string{`{"enum":["ready"]}`, `{"properties":{"name":{"type":"string"}},"required":["name"]}`, `{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]}]}`} {
		value := `[{"name":"outer"}]`
		if strings.Contains(target, `"enum"`) {
			value = `["ready"]`
		}
		root := loadComponentDocument(t, `{"Value":`+target+`,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias","nullable":false}],"default":`+value+`}}`)
		before, _ := root.MarshalJSON()
		a := analyzer{root: root, options: Options{PackageName: "org.example.reftupledefaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
		ds, err := a.analyze(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal(err)
		}
		c := a.components["Rows"]
		node, err := a.componentDefaultNode(c)
		if err != nil || node.element.schema != a.components["Value"].Schema || node.element.ty.Kind != Named || node.element.ty.Name != "org.example.reftupledefaults.models.ValueDto" {
			t.Fatal("actual selected source/codec lost", node, err)
		}
		if _, err := node.value(get(c.Schema, "default"), c.Pointer+"/default"); err != nil {
			t.Fatal(err)
		}
		decision, err := a.arrayItemsOwnDefault(c.Schema, c.Pointer, 0, true)
		if err != nil {
			t.Fatal(err)
		}
		bad := *decision
		bad.Pointer = c.Pointer + "/items"
		if _, err := a.tupleReferenceDefaultNode(c.Schema, c.Pointer, &bad, 0); err == nil {
			t.Fatal("manufactured selected source path admitted")
		}
		bytes, _ := c.Schema.MarshalJSON()
		copyArray, failure := document.Load(bytes, document.JSON, "")
		if failure != nil {
			t.Fatal(failure)
		}
		if _, err := a.tupleReferenceDefaultNode(copyArray, c.Pointer, decision, 0); err == nil {
			t.Fatal("copied outer tuple acquired selected source authority")
		}
		prefix := get(c.Schema, "prefixItems")
		original := prefix.Items()[0]
		bytes, _ = original.MarshalJSON()
		copy, failure := document.Load(bytes, document.JSON, "")
		if failure != nil {
			t.Fatal(failure)
		}
		prefix.Items()[0] = copy
		if a.validateInternalModels(ds) == nil {
			t.Fatal("same-byte selected reference acquired source authority")
		}
		prefix.Items()[0] = original
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal(err)
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("tuple defaults rewrote original tree")
		}
	}
}

func TestTupleReferenceDefaultCannotReplaceRawDecision(t *testing.T) {
	for _, tuple := range []string{
		`{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"items":true,"default":[]}`,
		`{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"},{"type":"integer"}],"default":[]}`,
	} {
		root := loadComponentDocument(t, `{"Value":{"type":"string"},"Rows":`+tuple+`}`)
		a := analyzer{root: root, options: Options{PackageName: "org.example.reftupledefaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
		if _, err := a.analyze(root); err != nil {
			t.Fatal(err)
		}
		c := a.components["Rows"]
		forged := &arrayItemsDecision{Selected: get(c.Schema, "prefixItems").Items()[0], Pointer: c.Pointer + "/prefixItems/0"}
		if _, err := a.tupleReferenceDefaultNode(c.Schema, c.Pointer, forged, 0); err == nil {
			t.Fatal("manufactured typed selection overrode original raw tuple decision")
		}
	}
}
