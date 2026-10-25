package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestStandardFormRetainsActualFieldsAndSourceTree(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","body":{"required":true,"content":{"application/x-www-form-urlencoded":{"schema":{"type":"object","properties":{"uuid":{"type":"string","format":"uuid"},"date":{"type":"string","format":"date"},"instant":{"type":"string","format":"date-time"}},"required":["uuid"]}}}}}`)
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.standardform", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	body, err := a.requestBody(get(root, "body"), "/body", "Submit")
	if err != nil {
		t.Fatal(err)
	}
	model, ok := body.Models[0].(*Model)
	if !ok || len(model.Nested) != 0 || len(body.FormFields) != 3 {
		t.Fatal("form lost actual retained model")
	}
	for i, name := range []string{"kotlin.uuid.Uuid", "kotlinx.datetime.LocalDate", "kotlin.time.Instant"} {
		if body.FormFields[i].Type.WithoutRootNull().Name != name || &body.FormFields[i] != &model.Properties[i] {
			t.Fatal("form invented scalar or property binding", i)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("form rewrote original source")
	}
}
