package schema

import "testing"

func TestUUIDFactorySourceAndPermission(t *testing.T) {
	root := loadComponentDocument(t, `{"T":{"type":"string","format":"uuid","minLength":2,"default":"550e8400-e29b-41d4-a716-446655440000"},"M":{"$ref":"#/components/schemas/T","default":"550e8400-e29b-41d4-a716-446655440001"},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":"550e8400-e29b-41d4-a716-446655440002"}}`)
	before, _ := root.MarshalJSON()
	plan, _, err := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, value string }{{"TDto", `"550e8400-e29b-41d4-a716-446655440000"`}, {"MDto", `"550e8400-e29b-41d4-a716-446655440001"`}, {"NDto", `"550e8400-e29b-41d4-a716-446655440002"`}} {
		decl, ok := componentDeclaration(plan.Models, tc.name).(*ScalarAlias)
		if !ok || decl.Scalar != "kotlin.uuid.Uuid" || decl.ComponentDefault != tc.value {
			t.Fatal("UUID String mapping/independent ownership drift", tc, decl)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("UUID factory mutated original source")
	}
	marked := get(get(get(root, "components"), "schemas"), "T")
	a := &analyzer{options: Options{Validation: true}}
	if err := a.markComponentDefaultSource(marked, "/components/schemas/T"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.boundedPrimitiveType(marked, "/components/schemas/T", 0); err != nil {
		t.Fatal("original UUID source denied", err)
	}
	clone := loadComponentDocument(t, `{"T":{"type":"string","format":"uuid","minLength":2,"default":"550e8400-e29b-41d4-a716-446655440000"}}`)
	copied := get(get(get(clone, "components"), "schemas"), "T")
	if _, _, err := a.boundedPrimitiveType(copied, "/components/schemas/T", 0); err == nil || err.Code != "E_SCHEMA_UNSUPPORTED" {
		t.Fatal("UUID factory permission leaked to copied source", err)
	}
}
