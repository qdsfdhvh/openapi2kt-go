package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestNestedStringEnumBoundsOriginalTreePermission(t *testing.T) {
	data := []byte(`{"type":"array","items":{"additionalProperties":{"enum":["one","x"],"minLength":2}},"default":[{"a/~":"one"}]}`)
	root, failure := document.Load(data, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	leaf := get(get(root, "items"), "additionalProperties")
	original, _ := root.MarshalJSON()
	if nestedStringEnumBoundsLeaf(root) != leaf {
		t.Fatal("selector lost original inline terminal")
	}
	a := objectDefaultTestAnalyzer(true)
	var declarations []Declaration
	node, err := a.nestedDefaultSchemaWithEnumBounds(root, "Tree", "/source/tree", &declarations, map[string]bool{}, 1, true, leaf)
	if err != nil || len(declarations) != 1 {
		t.Fatal("nested checked tree did not retain actual lexical declaration", err)
	}
	terminal := node.element.element
	if node.schema != root || terminal.schema != leaf || terminal.ty.Name != declarations[0].Name() || terminal.ty.Validation == nil || !terminal.stringEnum || !node.hasEnum() {
		t.Fatal("nested checked tree lost original source/type/check binding")
	}
	if a.validationSources[terminal.ty.Validation] != "/source/tree/items/additionalProperties" {
		t.Fatal("nested checked tree lost original constraint Pointer")
	}
	bad, loadError := document.Load([]byte(`{"value":[{"a/~":"x"}]}`), document.JSON, "")
	if loadError != nil {
		t.Fatal(loadError)
	}
	if _, err := node.value(get(bad, "value"), "/consumer/default"); err == nil || err.Pointer == nil || *err.Pointer != "/consumer/default/0/a~1~0" {
		t.Fatal("nested bound lost consuming annotation Pointer", err)
	}
	copy, _ := document.Load(data, document.JSON, "")
	for _, permission := range []*document.Value{nil, get(get(copy, "items"), "additionalProperties"), root} {
		if _, err := a.nestedDefaultSchemaWithEnumBounds(root, "Tree", "/source/tree", nil, nil, 1, true, permission); err == nil {
			t.Fatal("non-original terminal acquired scoped tree permission")
		}
	}
	actual, _ := root.MarshalJSON()
	if string(actual) != string(original) {
		t.Fatal("planning mutated original source")
	}
}
