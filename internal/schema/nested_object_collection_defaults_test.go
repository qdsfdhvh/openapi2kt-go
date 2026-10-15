package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func objectCollectionTestAnalyzer(checked bool) *analyzer {
	return &analyzer{options: Options{Validation: checked}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}}
}

func TestNestedObjectCollectionDefaultSourceAuthority(t *testing.T) {
	input := []byte(`{"type":"array","items":{"type":"object","required":["name"],"properties":{"name":{"type":"string"},"age":{"type":"integer","default":7}}},"default":[{"name":"a"},{"age":3,"name":"b"}]}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	original, _ := root.MarshalJSON()
	a := objectCollectionTestAnalyzer(true)
	node, failure := a.objectDefaultNode(root, "/field", 1, true)
	if failure != nil {
		t.Fatal(failure)
	}
	if node.ty.Kind != List || !node.element.object || node.element.schema != get(root, "items") {
		t.Fatal("container/object element did not borrow original nodes", node.ty, node.element)
	}
	if node.element.objectFields[0].node.schema != get(get(root, "items"), "properties").Members()[0].Value {
		t.Fatal("property node did not borrow original schema")
	}
	canonical, failure := node.value(get(root, "default"), "/field/default")
	if failure != nil || canonical != `[{"name":"a"},{"age":3,"name":"b"}]` {
		t.Fatal(canonical, failure)
	}
	// Element schemas cannot own the default annotation.
	items := get(root, "items")
	if _, err := a.objectDefaultChild(items, "/field/items", 2); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(original) != string(after) {
		t.Fatal("rewrote original source")
	}
}

func TestNestedObjectCollectionDefaultElementOwnDefaultRejected(t *testing.T) {
	items, err := document.Load([]byte(`{"type":"object","properties":{"n":{"type":"integer"}},"default":{"n":1}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := objectCollectionTestAnalyzer(false)
	if _, failure := a.objectDefaultChild(items, "/field/items", 2); failure == nil {
		t.Fatal("element own default admitted")
	}
}

func TestNestedObjectCollectionDefaultTypePlanIgnoresOwnAnnotation(t *testing.T) {
	withDefault := []byte(`{"type":"array","nullable":true,"items":{"type":"object","properties":{"a":{"type":"string"}}},"default":[{"a":"x"}]}`)
	without, err := document.Load([]byte(`{"type":"array","nullable":true,"items":{"type":"object","properties":{"a":{"type":"string"}}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := objectCollectionTestAnalyzer(false)
	annotated, err := document.Load(withDefault, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a.objectDefaultSchemas = map[*document.Value]bool{annotated: true}
	var nested, plain []Declaration
	ty, _, failure := a.modelPropertyType(annotated, "Value", "/field", &nested, map[string]bool{}, 1)
	if failure != nil {
		t.Fatal(failure)
	}
	b := objectCollectionTestAnalyzer(false)
	tyWithout, _, failure := b.modelPropertyType(without, "Value", "/field", &plain, map[string]bool{}, 1)
	if failure != nil {
		t.Fatal(failure)
	}
	if ty.WithoutRootNull().Kind != List || tyWithout.WithoutRootNull().Kind != List || len(nested) != 1 || len(plain) != 1 {
		t.Fatal("annotation changed the collection type plan", ty, tyWithout)
	}
}
