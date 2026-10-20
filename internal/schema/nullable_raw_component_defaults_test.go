package schema

import "testing"

func TestNullableRawComponentDefaultsRetainSourceAndCarrier(t *testing.T) {
	root := loadComponentDocument(t, `{"Free":{"type":"object","nullable":true,"default":null},"Open":{"type":["object","null"],"additionalProperties":true,"default":null}}`)
	before, err := root.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		plan, _, failure := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked})
		if failure != nil {
			t.Fatal(failure)
		}
		free, ok := componentDeclaration(plan.Models, "FreeDto").(*Model)
		if !ok || free.ComponentDefault != "null" || !free.Nullable || len(free.Properties) != 0 || len(free.Constants) != 0 {
			t.Fatal("nullable raw model backing changed", free)
		}
		open, ok := componentDeclaration(plan.Models, "OpenDto").(*ValueAlias)
		if !ok || open.ComponentDefault != "null" || open.Type.Kind != Nullable || open.Type.Element.Kind != JSONObject {
			t.Fatal("nullable raw dictionary backing changed", open)
		}
		after, err := root.MarshalJSON()
		if err != nil || string(before) != string(after) {
			t.Fatal("nullable raw factory mutated its original source", err)
		}
	}
}
