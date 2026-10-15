package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestNativeEnumBoundsBorrowOriginalVocabularyAndMetadata(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","enum":[1,2,3,4],"minimum":2,"multipleOf":2},"Alias":{"$ref":"#/components/schemas/Code"},"Record":{"properties":{"value":{"$ref":"#/components/schemas/Alias"}}}}}}`)
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	plan, _, err := Analyze(root, Options{PackageName: "org.example.enums", Validation: true})
	if err != nil {
		t.Fatal(err)
	}
	var checks *PrimitiveValidation
	for _, d := range plan.Models {
		if m, ok := d.(*Model); ok && m.ClassName == "Record" {
			checks = bare(m.Properties[0].Type).Validation
		}
	}
	if checks == nil || len(checks.Enum) != 4 || checks.Minimum.Literal != "2" || checks.MultipleOf.Literal != "2" || !plan.EmissionHints[checks] {
		t.Fatal(checks)
	}
	bytes, _ := root.MarshalJSON()
	if string(bytes) != string(input) {
		t.Fatal("source rewritten")
	}
}
