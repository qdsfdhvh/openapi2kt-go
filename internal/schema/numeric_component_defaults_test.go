package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestNumericFactorySourceAndPermission(t *testing.T) {
	root := loadComponentDocument(t, `{"T":{"type":"number","format":"float","minimum":0,"default":0.300000004},"M":{"$ref":"#/components/schemas/T","nullable":true,"default":null},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":0.5}}`)
	before, _ := root.MarshalJSON()
	plan, _, err := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true})
	if err != nil {
		t.Fatal(err)
	}
	target, ok := componentDeclaration(plan.Models, "TDto").(*ScalarAlias)
	if !ok || target.Scalar != "kotlin.Float" || target.ComponentDefault != "0.300000004" {
		t.Fatal("numeric root mapping/token drift", target)
	}
	outer, ok := componentDeclaration(plan.Models, "NDto").(*ValueAlias)
	if !ok || outer.Type.Kind == Nullable || outer.ComponentDefault != "0.5" {
		t.Fatal("numeric alias null intent/token drift", outer)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("numeric default mutated original sources")
	}
	source := get(get(get(root, "components"), "schemas"), "T")
	a := &analyzer{componentDefaultSchemas: map[*document.Value]bool{source: true}}
	if _, err := scalarTypeFor(a, "number", source, "/components/schemas/T"); err == nil {
		// The bound still needs its own validation role; marker permission is
		// neither arbitrary scalar-key permission nor a validation opt-in.
		t.Fatal("scalar permission bypassed bounded source grammar")
	}
	plain := loadComponentDocument(t, `{"T":{"type":"integer","format":"int64","default":9223372036854775807}}`)
	marked := get(get(get(plain, "components"), "schemas"), "T")
	a.componentDefaultSchemas[marked] = true
	if scalar, err := scalarTypeFor(a, "integer", marked, "/components/schemas/T"); err != nil || scalar != "kotlin.Long" {
		t.Fatal("original numeric source denied", scalar, err)
	}
	clone := loadComponentDocument(t, `{"T":{"type":"integer","format":"int64","default":9223372036854775807}}`)
	copied := get(get(get(clone, "components"), "schemas"), "T")
	if _, err := scalarTypeFor(a, "integer", copied, "/components/schemas/T"); err == nil || err.Code != "E_SCHEMA_UNSUPPORTED" {
		t.Fatal("numeric factory permission leaked to copied source", err)
	}
}
