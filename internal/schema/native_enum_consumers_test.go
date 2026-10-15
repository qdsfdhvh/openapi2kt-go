package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestNativeEnumCatalogChecksBorrowOriginalSourceThroughAliases(t *testing.T) {
	root, e := document.Load([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","enum":[1,2]},"Alias":{"$ref":"#/components/schemas/Code"},"Record":{"properties":{"code":{"$ref":"#/components/schemas/Alias"}}}}}}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	plan, _, err := Analyze(root, Options{PackageName: "org.example.enums", Validation: true})
	if err != nil {
		t.Fatal(err)
	}
	var checks *PrimitiveValidation
	for _, decl := range plan.Models {
		if model, ok := decl.(*Model); ok && model.ClassName == "Record" {
			checks = bare(model.Properties[0].Type).Validation
		}
	}
	if checks == nil || len(checks.Enum) != 2 || checks.Enum[0] != "1" || !plan.EmissionHints[checks] {
		t.Fatal(checks)
	}
	bytes, _ := root.MarshalJSON()
	if string(bytes) != `{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","enum":[1,2]},"Alias":{"$ref":"#/components/schemas/Code"},"Record":{"properties":{"code":{"$ref":"#/components/schemas/Alias"}}}}}}` {
		t.Fatal("rewrote original schema")
	}
}
