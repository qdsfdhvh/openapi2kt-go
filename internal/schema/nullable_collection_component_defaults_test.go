package schema

import "testing"

func TestNullableCollectionLeafFactoriesRetainSourceAndTypes(t *testing.T) {
	root := loadComponentDocument(t, `{"Rows":{"type":"array","items":{"type":["string","null"]},"default":[null,"x"]},"Bags":{"type":["object","null"],"additionalProperties":{"type":"integer","nullable":true},"default":{"z":7,"a":null}},"Target":{"type":"array","items":{"type":"string","nullable":true}},"Wrapped":{"$ref":"#/components/schemas/Target","default":[null,"w"]}}`)
	before, err := root.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		plan, _, failure := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked})
		if failure != nil {
			t.Fatal(failure)
		}
		rows, ok := componentDeclaration(plan.Models, "RowsDto").(*ValueAlias)
		if !ok || rows.ComponentDefault != `[null,"x"]` || rows.Type.Kind != List || rows.Type.Element.Kind != Nullable || rows.Type.Element.Element.Name != "kotlin.String" {
			t.Fatal("nullable list leaf backing changed", rows)
		}
		bags, ok := componentDeclaration(plan.Models, "BagsDto").(*ValueAlias)
		if !ok || bags.ComponentDefault != `{"a":null,"z":7}` || bags.Type.Kind != Nullable || bags.Type.Element.Kind != Map || bags.Type.Element.Element.Kind != Nullable || bags.Type.Element.Element.Element.Name != "kotlin.Int" {
			t.Fatal("nullable map leaf backing or canonical order changed", bags)
		}
		wrapped, ok := componentDeclaration(plan.Models, "WrappedDto").(*ValueAlias)
		if !ok || wrapped.ComponentDefault != `[null,"w"]` {
			t.Fatal("wrapper lost its own factory", wrapped)
		}
		target, ok := componentDeclaration(plan.Models, "TargetDto").(*ValueAlias)
		if !ok || target.ComponentDefault != "" {
			t.Fatal("wrapper changed its target", target)
		}
		after, err := root.MarshalJSON()
		if err != nil || string(before) != string(after) {
			t.Fatal("nullable collection leaf factory changed original source", err)
		}
	}
}
