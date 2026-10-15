package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestEnumDefaultsSourceOwnershipAndLockIdentity(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"value":{"enum":["ready","busy"],"default":"ready"}}}}}}`)
	root, err := document.ParseObject(input, len(input))
	if err != nil {
		t.Fatal(err)
	}
	original, _ := root.MarshalJSON()
	plan, warnings, failure := Analyze(root, Options{PackageName: "org.example.defaults"})
	after, _ := root.MarshalJSON()
	if failure != nil || len(warnings) != 0 || len(plan.Models) != 1 || string(after) != string(original) {
		t.Fatal("source changed", warnings, failure)
	}
	model, ok := plan.Models[0].(*Model)
	if !ok || len(model.Properties) != 1 || model.Properties[0].Default != `Value.fromSerialName("ready")` {
		t.Fatal(plan)
	}
	first := nameLockShapeV1(model, "org.example.defaults")
	model.Properties[0].Default = `Value.fromSerialName("busy")`
	if first == nameLockShapeV1(model, "org.example.defaults") {
		t.Fatal("lock lost new default")
	}
	model.Properties[0].Default = ""
	legacy := nameLockShapeV1(model, "org.example.defaults")
	if first == legacy {
		t.Fatal("default collided with legacy identity")
	}
	model.Properties[0].Default = `Value.fromSerialName("ready")`
	if first != nameLockShapeV1(model, "org.example.defaults") {
		t.Fatal("default fingerprint nondeterministic")
	}
}
