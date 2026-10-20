package schema

import "testing"

func TestExplicitNullableWrapperDefaultsKeepOriginalSource(t *testing.T) {
	root := loadComponentDocument(t, `{"T":{"allOf":[{"properties":{"z":{"type":"string"}},"required":["z"]}]},"M":{"$ref":"#/components/schemas/T"},"N":{"$ref":"#/components/schemas/M","nullable":true,"default":null}}`)
	before, _ := root.MarshalJSON()
	for _, checked := range []bool{false, true} {
		plan, _, failure := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked})
		if failure != nil {
			t.Fatal(failure)
		}
		alias, ok := componentDeclaration(plan.Models, "NDto").(*ValueAlias)
		if !ok || alias.ComponentDefault != "null" || alias.Type.Kind != Nullable {
			t.Fatal("nullable alias lost factory/type", alias)
		}
		if target, ok := componentDeclaration(plan.Models, "TDto").(*Model); !ok || target.ComponentDefault != "" || target.Nullable {
			t.Fatal("wrapper changed terminal", target)
		}
		if middle, ok := componentDeclaration(plan.Models, "MDto").(*ValueAlias); !ok || middle.ComponentDefault != "" || middle.Type.Kind == Nullable {
			t.Fatal("wrapper changed intermediate alias", middle)
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("nullable wrapper rewrote source")
		}
	}
}
