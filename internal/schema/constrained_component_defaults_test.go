package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestConstrainedScalarFactoryKeepsSourceAndChecks(t *testing.T) {
	root := loadComponentDocument(t, `{"T":{"type":"string","minLength":2,"maxLength":6,"pattern":"^[a-z]+$","default":"given"},"M":{"$ref":"#/components/schemas/T","nullable":true,"default":null},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":"outer"}}`)
	before, _ := root.MarshalJSON()
	plan, _, err := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true})
	if err != nil {
		t.Fatal(err)
	}
	target, ok := componentDeclaration(plan.Models, "TDto").(*ScalarAlias)
	if !ok || target.Scalar != "kotlin.String" || target.ComponentDefault != `"given"` {
		t.Fatal("bounded scalar changed type/factory", target)
	}
	middle, ok := componentDeclaration(plan.Models, "MDto").(*ValueAlias)
	if !ok || middle.Type.Kind != Nullable || middle.ComponentDefault != "null" {
		t.Fatal("nullable bounded alias lost own value", middle)
	}
	outer, ok := componentDeclaration(plan.Models, "NDto").(*ValueAlias)
	if !ok || outer.Type.Kind == Nullable || outer.Type.Validation == nil || outer.ComponentDefault != `"outer"` {
		t.Fatal("narrowed bounded alias lost checks/value", outer)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("factory modified original constraint/default sources")
	}
}

func TestBoundedComponentDefaultPermissionRequiresOriginalNode(t *testing.T) {
	root := loadComponentDocument(t, `{"T":{"type":"string","minLength":2,"default":"given"}}`)
	source := get(get(get(root, "components"), "schemas"), "T")
	a := &analyzer{options: Options{Validation: true}, componentDefaultSchemas: map[*document.Value]bool{source: true}}
	if _, _, err := a.boundedPrimitiveType(source, "/components/schemas/T", 0); err != nil {
		t.Fatal("marked original rejected", err)
	}
	clone := loadComponentDocument(t, `{"T":{"type":"string","minLength":2,"default":"given"}}`)
	copied := get(get(get(clone, "components"), "schemas"), "T")
	if _, _, err := a.boundedPrimitiveType(copied, "/components/schemas/T", 0); err == nil || err.Code != "E_SCHEMA_UNSUPPORTED" {
		t.Fatal("default permission leaked to copied source", err)
	}
}
