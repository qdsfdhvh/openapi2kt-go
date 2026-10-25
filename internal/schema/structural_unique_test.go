package schema

import (
	"encoding/json"
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestStructuralUniqueDefaultKeyEscapingIsBounded(t *testing.T) {
	encoded, _ := json.Marshal(strings.Repeat(`\`, 64))
	value := string(encoded)
	ty := TypeRef{Kind: Scalar, Name: "kotlin.String"}
	for range 8 {
		value = "[" + value + "]"
		child := ty
		ty = TypeRef{Kind: List, Element: &child}
	}
	input := []byte(`{"value":` + value + `}`)
	root, err := document.ParseObject(input, len(input))
	if err != nil {
		t.Fatal(err)
	}
	key := structuralUniqueDefaultKey(ty, get(root, "value"))
	if len(key) > len(input)*8 {
		t.Fatal("nested keys repeatedly escape their already encoded children", len(key), len(input))
	}
}

func TestStructuralUniqueSourceTypeAndLockIdentity(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"value":{"type":"array","uniqueItems":true,"items":{"type":"array","items":{"type":"integer","enum":[1,2]}}}}}}}}`)
	root, err := document.ParseObject(input, len(input))
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	plan, _, failure := Analyze(root, Options{PackageName: "org.example", Validation: true})
	if failure != nil {
		t.Fatal(failure)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("structural uniqueness changed original source")
	}
	model := plan.Models[0].(*Model)
	ty := model.Properties[0].Type.WithoutRootNull()
	source := get(get(get(get(get(root, "components"), "schemas"), "Record"), "properties"), "value")
	pointer := "/components/schemas/Record/properties/value/uniqueItems"
	uses := []structuralUniqueUse{{ty.Element, pointer, get(source, "items"), "/components/schemas/Record/properties/value/items"}}
	a := &analyzer{structuralUniqueUses: &uses}
	if err := a.validateStructuralUniqueUses(); err != nil {
		t.Fatal(err)
	}
	leaf := ty.Element.Element
	checks := leaf.Validation
	leaf.Validation = nil
	if err := a.validateStructuralUniqueUses(); err == nil || *err.Pointer != pointer {
		t.Fatal("projected enum without retained membership gained structural equality", err)
	}
	leaf.Validation = checks
	shape := nameLockShapeV1(model, "org.example")
	ty.Collection.Unique = false
	if shape == nameLockShapeV1(model, "org.example") {
		t.Fatal("structural check lost persisted shape identity")
	}
	if structuralUniqueType(TypeRef{Kind: Scalar, Name: "kotlin.collections.List"}, 0) {
		t.Fatal("spelling granted structural equality")
	}
}
