package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestJSONUniqueOriginalNodeAndLockIdentity(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"value":{"type":"array","items":true,"uniqueItems":true}}}}}}`)
	root, err := document.ParseObject(input, len(input))
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	plan, _, failure := Analyze(root, Options{PackageName: "org.example", Validation: true})
	after, _ := root.MarshalJSON()
	if failure != nil || string(before) != string(after) {
		t.Fatal("JSON source mutation", failure)
	}
	model := plan.Models[0].(*Model)
	ty := model.Properties[0].Type.WithoutRootNull()
	if ty.Collection == nil || !ty.Collection.UniqueJSON || ty.Collection.Unique {
		t.Fatal("JSON/native policy collapsed", ty)
	}
	first := nameLockShapeV1(model, "org.example")
	ty.Collection.UniqueJSON = false
	if first == nameLockShapeV1(model, "org.example") {
		t.Fatal("JSON uniqueness lost lock identity")
	}
	ty.Collection.UniqueJSON = true
	if first != nameLockShapeV1(model, "org.example") {
		t.Fatal("JSON uniqueness unstable identity")
	}
	if !strings.Contains(string(before), `"items":true`) {
		t.Fatal("fixture did not retain literal source")
	}
}
