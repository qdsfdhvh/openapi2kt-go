package schema

import "testing"

func TestNestedComponentDefaultFactoriesKeepSourceAndAliasTypes(t *testing.T) {
	root := loadComponentDocument(t, `{"N":{"type":["object","null"],"additionalProperties":{"type":"array","nullable":true,"items":{"type":"integer","nullable":true}},"default":{"z":[1,null],"a":null}},"Target":{"additionalProperties":{"additionalProperties":{"type":"string"}}},"Wrapped":{"$ref":"#/components/schemas/Target","default":{"z":{"z":"last","a":"first"},"a":{"/~":"key","$ref":"data"}}}}`)
	before, err := root.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		plan, _, failure := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked})
		if failure != nil {
			t.Fatal(failure)
		}
		alias, ok := componentDeclaration(plan.Models, "NDto").(*ValueAlias)
		if !ok || alias.ComponentDefault != `{"a":null,"z":[1,null]}` || alias.Type.Kind != Nullable || alias.Type.Element.Kind != Map {
			t.Fatal("outer alias lost its nullable type/canonical value", alias)
		}
		inner := alias.Type.Element.Element
		if inner.Kind != Nullable || inner.Element.Kind != List || inner.Element.Element.Kind != Nullable || inner.Element.Element.Element.Name != "kotlin.Int" {
			t.Fatal("nested alias nullability was flattened", inner)
		}
		wrapped, ok := componentDeclaration(plan.Models, "WrappedDto").(*ValueAlias)
		if !ok || wrapped.ComponentDefault != `{"a":{"$ref":"data","/~":"key"},"z":{"a":"first","z":"last"}}` {
			t.Fatal("nested wrapper factory lost canonical order/opaque keys", wrapped)
		}
		target, ok := componentDeclaration(plan.Models, "TargetDto").(*ValueAlias)
		if !ok || target.ComponentDefault != "" {
			t.Fatal("nested wrapper factory changed its target", target)
		}
		after, err := root.MarshalJSON()
		if err != nil || string(before) != string(after) {
			t.Fatal("nested component factory rewrote original schema/value nodes", err)
		}
	}
}
