package schema

import "testing"

func TestRawJSONWrapperDefaultsRetainOriginalSourceAndTargets(t *testing.T) {
	root := loadComponentDocument(t, `{"Free":{"type":"object","nullable":true},"Open":{"type":["object","null"],"additionalProperties":true},"WrappedFree":{"$ref":"#/components/schemas/Free","default":null},"WrappedOpen":{"$ref":"#/components/schemas/Open","default":{"z":[true,null],"a":{"$ref":"data"}}}}`)
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
		if !ok || free.ComponentDefault != "" || !free.Nullable || len(free.Properties) != 0 {
			t.Fatal("raw wrapper changed its free target", free)
		}
		open, ok := componentDeclaration(plan.Models, "OpenDto").(*ValueAlias)
		if !ok || open.ComponentDefault != "" || open.Type.Kind != Nullable || open.Type.Element.Kind != JSONObject {
			t.Fatal("raw wrapper changed its open target", open)
		}
		for _, tc := range []struct{ name, encoded string }{
			{"WrappedFreeDto", "null"},
			{"WrappedOpenDto", `{"z":[true,null],"a":{"$ref":"data"}}`},
		} {
			wrapped, ok := componentDeclaration(plan.Models, tc.name).(*ValueAlias)
			if !ok || wrapped.ComponentDefault != tc.encoded {
				t.Fatal("wrapper lost own annotation/raw order/opaque keys", wrapped)
			}
		}
		after, err := root.MarshalJSON()
		if err != nil || string(before) != string(after) {
			t.Fatal("raw wrapper factory rewrote source nodes", err)
		}
	}
}
