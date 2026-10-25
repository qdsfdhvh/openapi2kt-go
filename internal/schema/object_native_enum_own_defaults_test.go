package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestObjectNativeEnumOwnDefaultOriginalSourcesAndRoles(t *testing.T) {
	root := loadComponentDocument(t, `{"Target":{"properties":{"value":{"type":"integer","format":"int32","nullable":true,"enum":[1,2],"default":null}},"default":{}},"Alias":{"$ref":"#/components/schemas/Target","nullable":true,"default":null}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.native", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	if _, err := a.analyze(root); err != nil {
		t.Fatal(err)
	}
	target := a.referenceWrapperDefaultTarget(a.components["Alias"])
	node, err := a.wrapperShapeDefaultNode(target)
	if err != nil {
		t.Fatal(err)
	}
	original := get(get(target.Schema, "properties"), "value")
	pointer := "/components/schemas/Target/properties/value"
	child := node.objectFields[0].node
	if node.schema != target.Schema || child.schema != original || child.pointer != pointer || child.native == nil || child.ty.Kind != Nullable || child.ty.WithoutRootNull().Name != "kotlin.Int" || a.validationSources[child.ty.WithoutRootNull().Validation] != pointer {
		t.Fatal("native own default lost original schema, type, nullable or validation authority", node)
	}
	for _, spare := range []int{2, 1} {
		work := maxNestedDefaultNodes - spare
		a.objectDefaultWork = &work
		_, err := a.wrapperShapeDefaultNode(target)
		if spare == 2 && (err != nil || work != maxNestedDefaultNodes) || spare == 1 && (err == nil || err.Code != "E_MODEL_LIMIT") {
			t.Fatal("native own default reset shared source work", work, err)
		}
	}
	a.objectDefaultWork = nil
	if _, err := a.objectDefaultPropertyNode(original, pointer, 1, false); err == nil || err.Pointer == nil || *err.Pointer != pointer+"/default" {
		t.Fatal("ordinary own annotation leaked selected-variant role", err)
	}
	array, parseErr := document.ParseObject([]byte(`{"type":"array","items":{"type":"boolean","enum":[true,false],"default":false},"default":[]}`), document.MaxDocumentBytes)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	if _, err := a.nestedDefaultSchema(array, "Probe", "/probe", nil, nil, 0, true); err == nil || err.Pointer == nil || *err.Pointer != "/probe/items/default" {
		t.Fatal("native own annotation leaked collection-element role", err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("native own defaults mutated source AST")
	}
}
