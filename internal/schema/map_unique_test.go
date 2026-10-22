package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestMapUniqueOriginalSourceAndIdentity(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"value":{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":{"type":"string","nullable":true,"minLength":1}}}}}}}}`)
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
		t.Fatal("Map uniqueness mutated original source")
	}
	model := plan.Models[0].(*Model)
	ty := model.Properties[0].Type.WithoutRootNull()
	if ty.Collection == nil || !ty.Collection.Unique || ty.Collection.UniqueJSON || !uniqueMapType(*ty.Element) {
		t.Fatal("Map equality policy lost", ty)
	}
	first := nameLockShapeV1(model, "org.example")
	ty.Collection.Unique = false
	if first == nameLockShapeV1(model, "org.example") {
		t.Fatal("Map equality lost lock identity")
	}
	if uniqueMapType(TypeRef{Kind: Scalar, Name: "kotlin.collections.Map"}) {
		t.Fatal("scalar spelling granted Map authority")
	}
}

func TestMapSpecialUniqueSourceAndIdentity(t *testing.T) {
	for _, format := range []string{"uuid", "date", "date-time"} {
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"value":{"type":"array","uniqueItems":true,"items":{"type":"object","nullable":true,"additionalProperties":{"type":"string","format":"` + format + `","nullable":true}}}}}}}}`)
		root, failure := document.ParseObject(input, len(input))
		if failure != nil {
			t.Fatal(failure)
		}
		before, _ := root.MarshalJSON()
		plan, _, err := Analyze(root, Options{PackageName: "org.example", Validation: true})
		if err != nil {
			t.Fatal(format, err)
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("mutated original source", format)
		}
		model := plan.Models[0].(*Model)
		ty := model.Properties[0].Type.WithoutRootNull()
		if ty.Collection == nil || !ty.Collection.Unique || ty.Collection.UniqueJSON || !uniqueMapType(*ty.Element) {
			t.Fatal("equality identity lost", format)
		}
		first := nameLockShapeV1(model, "org.example")
		ty.Collection.Unique = false
		if first == nameLockShapeV1(model, "org.example") {
			t.Fatal("lock check lost", format)
		}
		if !strings.Contains(string(before), `"format":"`+format+`"`) {
			t.Fatal("source format lost")
		}
	}
}
