package schema

import "testing"

func TestComponentNullFactoriesRetainSourceAndBacking(t *testing.T) {
	root := loadComponentDocument(t, `{"Bytes":{"type":"string","format":"byte","nullable":true,"default":null},"Mode":{"type":["string","null"],"enum":["a","b"],"default":null},"Rows":{"type":"array","nullable":true,"items":{"type":"integer"},"default":null}}`)
	before, err := root.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	plan, _, failure := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto"})
	if failure != nil {
		t.Fatal(failure)
	}
	bytes, ok := componentDeclaration(plan.Models, "BytesDto").(*ValueAlias)
	if !ok || bytes.ComponentDefault != "null" || bytes.Type.Kind != Nullable || bytes.Type.Element.Kind != Scalar || bytes.Type.Element.Name != "kotlin.ByteArray" {
		t.Fatal("byte alias backing drift", bytes)
	}
	rows, ok := componentDeclaration(plan.Models, "RowsDto").(*ValueAlias)
	if !ok || rows.ComponentDefault != "null" || rows.Type.Kind != Nullable || rows.Type.Element.Kind != List || rows.Type.Element.Element.Name != "kotlin.Int" {
		t.Fatal("list backing drift", rows)
	}
	mode, ok := componentDeclaration(plan.Models, "ModeDto").(*StringEnum)
	if !ok || mode.ComponentDefault != "null" || len(mode.Members) != 2 {
		t.Fatal("enum vocabulary/classifier drift", mode)
	}
	after, err := root.MarshalJSON()
	if err != nil || string(before) != string(after) {
		t.Fatal("null factory changed original document", err)
	}
}
