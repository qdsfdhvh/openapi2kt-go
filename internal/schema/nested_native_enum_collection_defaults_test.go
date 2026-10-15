package schema

import (
	"reflect"
	"testing"

	"openapi-kt/internal/document"
)

func TestNestedNativeEnumDefaultOriginalNodeAndCheckAuthority(t *testing.T) {
	for _, checked := range []bool{false, true} {
		input := []byte(`{"type":"array","items":{"type":"object","nullable":true,"additionalProperties":{"type":["number","null"],"format":"float","enum":[0.5,1]}},"default":[{"z":null,"a":0.5},null]}`)
		root, err := document.Load(input, document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		original, _ := root.MarshalJSON()
		a := analyzer{options: Options{Validation: checked}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}}
		node, failure := a.nestedDefaultSchema(root, "Value", "/field", nil, nil, 1, true)
		if failure != nil {
			t.Fatal(checked, failure)
		}
		leaf := node.element.element
		schema := get(get(root, "items"), "additionalProperties")
		if node.schema != root || node.element.schema != get(root, "items") || leaf.schema != schema || leaf.native != &a || leaf.pointer != "/field/items/additionalProperties" || leaf.enum != nil {
			t.Fatal("copied source or lexical enum authority", node)
		}
		ty := leaf.ty.WithoutRootNull()
		if ty.Kind != Scalar || ty.Name != "kotlin.Float" || ty.Serializer != "" || leaf.ty.Kind != Nullable || node.ty.Kind != List || node.element.ty.Kind != Nullable {
			t.Fatal("scalar/nearest nullable authority lost", node.ty, leaf.ty)
		}
		if checked {
			if ty.Validation == nil || !reflect.DeepEqual(ty.Validation.Enum, []string{"0.5f", "1.0f"}) || a.validationSources[ty.Validation] != leaf.pointer || !a.emissionHints[ty.Validation] || len(a.warnings) != 0 {
				t.Fatal(ty.Validation, a.warnings)
			}
		} else if ty.Validation != nil || len(a.warnings) != 1 || a.warnings[0].Pointer == nil || *a.warnings[0].Pointer != leaf.pointer+"/enum" {
			t.Fatal(ty.Validation, a.warnings)
		}
		canonical, failure := node.value(get(root, "default"), "/field/default")
		if failure != nil || canonical != `[{"a":0.5,"z":null},null]` {
			t.Fatal(canonical, failure)
		}
		bad, err := document.Load([]byte(`{"value":[{"a":1.000000001}]}`), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, failure := node.value(get(bad, "value"), "/field/default"); failure == nil || failure.Pointer == nil || *failure.Pointer != "/field/default/0/a" {
			t.Fatal("rounded nonmember admitted", failure)
		}
		after, _ := root.MarshalJSON()
		if string(original) != string(after) {
			t.Fatal("rewrote original source")
		}
	}
}

func TestNestedNativeEnumDefaultReusesExistingBoundedLeafPlan(t *testing.T) {
	input := []byte(`{"type":"array","items":{"additionalProperties":{"type":["integer","null"],"enum":[null,2,4],"minimum":2,"maximum":4,"multipleOf":2}},"default":[{"a":null,"b":4}]}`)
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{options: Options{Validation: true}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}}
	node, failure := a.nestedDefaultSchema(root, "Value", "/field", nil, nil, 1, true)
	if failure != nil {
		t.Fatal(failure)
	}
	originalLeaf := get(get(root, "items"), "additionalProperties")
	direct, failure := a.collectionDefaultElement(originalLeaf, "ValueInnerValue", "/field/items/additionalProperties", nil, nil, 3)
	if failure != nil {
		t.Fatal(failure)
	}
	leaf := node.element.element
	if !reflect.DeepEqual(leaf.ty, direct) || leaf.schema != originalLeaf || leaf.native != &a || len(leaf.ty.WithoutRootNull().Validation.Enum) != 2 {
		t.Fatal("borrowed different leaf plan", leaf.ty, direct)
	}
	if _, failure := node.value(get(root, "default"), "/field/default"); failure != nil {
		t.Fatal(failure)
	}
	if leaf.ty.WithoutRootNull().Validation.Minimum == nil || leaf.ty.WithoutRootNull().Validation.MultipleOf == nil {
		t.Fatal("discarded original checks")
	}
}
