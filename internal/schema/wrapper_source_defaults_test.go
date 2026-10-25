package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestWrapperShapeDefaultBorrowsOriginalNodesAndOrder(t *testing.T) {
	for _, shape := range []string{
		`"type":["object","null"],"properties":{"z":{"type":"string","enum":["on","off"],"default":"on"},"a":{"type":["string","null"]}}`,
		`"nullable":true,"allOf":[{"properties":{"z":{"type":"string","enum":["on","off"],"default":"on"}}},{"properties":{"a":{"type":["string","null"]}}}]`,
	} {
		root := loadComponentDocument(t, `{"Target":{`+shape+`},"Alias":{"$ref":"#/components/schemas/Target","nullable":false,"default":{}}}`)
		before, _ := root.MarshalJSON()
		a := analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
		if _, err := a.analyze(root); err != nil {
			t.Fatal(err)
		}
		target := a.referenceWrapperDefaultTarget(a.components["Alias"])
		node, err := a.wrapperShapeDefaultNode(target)
		if err != nil || node.schema != a.components["Target"].Schema || node.pointer != "/components/schemas/Target" || node.ty.Kind == Nullable {
			t.Fatal("copied target or lost root-only nullable override", node, err)
		}
		properties := get(target.Schema, "properties")
		pointer := "/components/schemas/Target/properties/z"
		if has(target.Schema, "allOf") {
			properties = get(get(target.Schema, "allOf").Items()[0], "properties")
			pointer = "/components/schemas/Target/allOf/0/properties/z"
		}
		if len(node.objectFields) != 2 || node.objectFields[0].wire != "z" || node.objectFields[1].wire != "a" || node.objectFields[0].node.schema != get(properties, "z") || node.objectFields[0].node.pointer != pointer || node.objectFields[1].node.ty.Kind != Nullable {
			t.Fatal("lost original child nodes, order, pointers or nullability", node)
		}
		if _, err := a.objectDefaultPropertyNode(get(properties, "z"), pointer, 1, false); err == nil || err.Pointer == nil || *err.Pointer != pointer+"/default" {
			t.Fatal("ordinary property permission leaked into selected-variant member role", err)
		}
		for _, spare := range []int{3, 2} {
			work := maxNestedDefaultNodes - spare
			a.objectDefaultWork = &work
			_, err := a.wrapperShapeDefaultNode(target)
			if spare == 3 && (err != nil || work != maxNestedDefaultNodes) || spare == 2 && (err == nil || err.Code != "E_MODEL_LIMIT") {
				t.Fatal("wrapper reset or exceeded shared original source work", work, err)
			}
		}
		a.objectDefaultWork = nil
		// A property-own annotation never grants an element-own annotation.
		bad, parseErr := document.ParseObject([]byte(`{"type":"array","items":{"type":"string","enum":["on"],"default":"on"},"default":[]}`), document.MaxDocumentBytes)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		if _, err := a.nestedDefaultSchema(bad, "Probe", "/probe", nil, nil, 0, true); err == nil || err.Pointer == nil || *err.Pointer != "/probe/items/default" {
			t.Fatal("field own-default permission leaked to array element", err)
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("wrapper default changed original AST")
		}
	}
}
