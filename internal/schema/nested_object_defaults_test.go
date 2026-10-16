package schema

import (
	"reflect"
	"testing"

	"openapi-kt/internal/document"
)

func objectDefaultTestAnalyzer(checked bool) *analyzer {
	return &analyzer{options: Options{Validation: checked}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}}
}

func TestNestedObjectDefaultOriginalNodeAndValueAuthority(t *testing.T) {
	input := []byte(`{"type":"object","required":["name"],"properties":{"name":{"type":"string"},"status":{"type":"string","enum":["on","off"]},"child":{"type":"object","properties":{"flag":{"type":"boolean"}},"default":{"flag":true}},"rows":{"type":"array","items":{"type":"integer"}}},"default":{"rows":[1,2],"status":"on","name":"outer","child":{"flag":false}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	original, _ := root.MarshalJSON()
	a := objectDefaultTestAnalyzer(true)
	node, failure := a.objectDefaultNode(root, "/field", 1, true)
	if failure != nil {
		t.Fatal(failure)
	}
	if node.schema != root || !node.object || node.pointer != "/field" {
		t.Fatal("not borrowing the original field node", node)
	}
	properties := get(root, "properties")
	for i, field := range node.objectFields {
		if field.node.schema != properties.Members()[i].Value {
			t.Fatalf("field %d does not borrow its original property node", i)
		}
	}
	child := node.objectFields[2].node
	if !child.object || child.objectFields[0].node.schema != get(get(properties, "child"), "properties").Members()[0].Value {
		t.Fatal("nested object did not recurse on original nodes", child)
	}
	// Canonical encoding sorts keys; required member order is irrelevant.
	canonical, failure := node.value(get(root, "default"), "/field/default")
	if failure != nil || canonical != `{"child":{"flag":false},"name":"outer","rows":[1,2],"status":"on"}` {
		t.Fatal(canonical, failure)
	}
	// Unknown key, missing required and non-nullable null keep original value Pointers.
	for _, tc := range []struct{ value, pointer string }{
		{`{"name":"x","extra":1}`, "/field/default/extra"},
		{`{"status":"on"}`, "/field/default"},
		{`{"name":null}`, "/field/default/name"},
	} {
		value, err := document.Load([]byte(tc.value), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, failure := node.value(value, "/field/default"); failure == nil || failure.Pointer == nil || *failure.Pointer != tc.pointer {
			t.Fatalf("want failure at %s, got %v", tc.pointer, failure)
		}
	}
	after, _ := root.MarshalJSON()
	if string(original) != string(after) {
		t.Fatal("rewrote original source")
	}
}

func TestNestedObjectDefaultLeafChecksAndNullability(t *testing.T) {
	input := []byte(`{"type":"object","properties":{"kind":{"type":"string","enum":["a","b"]},"code":{"type":["integer","null"],"enum":[1,2]},"note":{"type":["string","null"]},"fixed":{"type":"string"}},"default":{"kind":"a","code":null,"note":null,"fixed":"x"}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		a := objectDefaultTestAnalyzer(checked)
		node, failure := a.objectDefaultNode(root, "/field", 1, true)
		if failure != nil {
			t.Fatal(checked, failure)
		}
		if _, failure := node.value(get(root, "default"), "/field/default"); failure != nil {
			t.Fatal(checked, failure)
		}
		// String enum membership uses the original vocabulary in both modes;
		// native membership is validation-only.
		bad, err := document.Load([]byte(`{"kind":"z","fixed":"x"}`), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, failure := node.value(bad, "/field/default"); failure == nil || failure.Pointer == nil || *failure.Pointer != "/field/default/kind" {
			t.Fatal("string enum membership not strict", checked, failure)
		}
		badNative, err := document.Load([]byte(`{"kind":"a","code":3,"fixed":"x"}`), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		// Default membership is strict against the original vocabulary in both
		// modes; only runtime wire enforcement is validation-gated.
		if _, failure := node.value(badNative, "/field/default"); failure == nil || failure.Pointer == nil || *failure.Pointer != "/field/default/code" {
			t.Fatal("native membership not strict", checked, failure)
		}
	}
	// Per-layer nullability stays independent of the outer object.
	narrow, err := document.Load([]byte(`{"type":"object","properties":{"fixed":{"type":"string"}},"default":{"fixed":null}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := objectDefaultTestAnalyzer(false)
	node, failure := a.objectDefaultNode(narrow, "/field", 1, true)
	if failure != nil {
		t.Fatal(failure)
	}
	if _, failure := node.value(get(narrow, "default"), "/field/default"); failure == nil || failure.Pointer == nil || *failure.Pointer != "/field/default/fixed" {
		t.Fatal("non-nullable property accepted null", failure)
	}
}

func TestNestedObjectDefaultTypePlanIgnoresOwnAnnotation(t *testing.T) {
	// The shared object type planner must ignore exactly this registered node's
	// own annotation and keep the same nested declaration plan.
	input := []byte(`{"type":"object","properties":{"a":{"type":"string"}},"default":{"a":"x"}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := objectDefaultTestAnalyzer(false)
	a.objectDefaultSchemas = map[*document.Value]bool{root: true}
	var nested []Declaration
	ty, _, failure := a.modelPropertyType(root, "Value", "/field", &nested, map[string]bool{}, 1)
	if failure != nil {
		t.Fatal(failure)
	}
	if ty.Kind != Named || ty.Name != "Value" || len(nested) != 1 {
		t.Fatal(ty, nested)
	}
	without, err := document.Load([]byte(`{"type":"object","properties":{"a":{"type":"string"}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	b := objectDefaultTestAnalyzer(false)
	var nestedWithout []Declaration
	tyWithout, _, failure := b.modelPropertyType(without, "Value", "/field", &nestedWithout, map[string]bool{}, 1)
	if failure != nil {
		t.Fatal(failure)
	}
	if !reflect.DeepEqual(ty, tyWithout) || len(nestedWithout) != 1 {
		t.Fatal("annotation changed the type plan", ty, tyWithout)
	}
}
