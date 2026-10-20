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

func TestAllOfComponentNullFactoryRetainsSourceAndConstructor(t *testing.T) {
	root := loadComponentDocument(t, `{"N":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}],"nullable":true,"default":null}}`)
	before, err := root.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		plan, _, failure := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked})
		if failure != nil {
			t.Fatal(failure)
		}
		model := componentModel(plan.Models, "NDto")
		if model == nil || model.ComponentDefault != "null" || !model.Nullable || len(model.Properties) != 2 {
			t.Fatal("allOf null factory changed its constructor", model)
		}
		if model.Properties[0].WireName != "a" || model.Properties[0].Default != "" || model.Properties[0].Type.Kind == Nullable || model.Properties[1].WireName != "b" {
			t.Fatal("allOf source order or required presence changed", model.Properties)
		}
		after, err := root.MarshalJSON()
		if err != nil || string(before) != string(after) {
			t.Fatal("allOf null factory changed original document", err)
		}
	}
}
