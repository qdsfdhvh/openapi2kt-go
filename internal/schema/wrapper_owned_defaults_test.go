package schema

import "testing"

func TestWrapperDefaultsRetainAllOriginalOwners(t *testing.T) {
	root := loadComponentDocument(t, `{"T":{"type":"array","items":{"type":["string","null"]},"default":["target"]},"M":{"$ref":"#/components/schemas/T","nullable":true,"default":null},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":[null,"outer"]},"P":{"$ref":"#/components/schemas/N"}}`)
	before, _ := root.MarshalJSON()
	for _, checked := range []bool{false, true} {
		plan, _, err := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked})
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name, value string
			nullable    bool
		}{{"TDto", `["target"]`, false}, {"MDto", "null", true}, {"NDto", `[null,"outer"]`, false}, {"PDto", "", false}} {
			alias, ok := componentDeclaration(plan.Models, tc.name).(*ValueAlias)
			if !ok || alias.ComponentDefault != tc.value || (alias.Type.Kind == Nullable) != tc.nullable {
				t.Fatal("independent annotation/type ownership lost", tc, alias)
			}
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("component defaults changed original source")
		}
	}
}
