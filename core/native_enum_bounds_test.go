package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumBoundsKeepVocabularyAndNumericConstraints(t *testing.T) {
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","enum":[1,2,3,4],"minimum":2,"multipleOf":2},"Record":{"properties":{"code":{"$ref":"#/components/schemas/Code"},"value":{"type":"integer","enum":[1,2,3,4],"minimum":2,"maximum":3,"default":2},"rows":{"type":"array","items":{"type":"number","enum":[1,2,3],"exclusiveMinimum":1,"maximum":2}}}}}}}`)
	r := request(doc, "Dto")
	r.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"does not belong to enum.", "value >= 2", "value % 2 == 0", "value > 1.0", "value <= 2.0", "value <= 3", "= 2"} {
		if !strings.Contains(source(t, got, "RecordDto"), part) {
			t.Fatal(part, source(t, got, "RecordDto"))
		}
	}
	again, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal(err)
	}
}
