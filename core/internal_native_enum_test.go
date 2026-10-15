package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInternalNativeEnumPointersRetainMappedVocabularyAndBounds(t *testing.T) {
	doc := schemas(`{"Bridge":{"$ref":"#/components/schemas/Source/properties/code"},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Bridge"},"direct":{"$ref":"#/components/schemas/Source/properties/code"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/code"}},"flag":{"$ref":"#/components/schemas/Source/properties/flag"}}},"Source":{"properties":{"code":{"type":"integer","enum":[1,2,3,4],"minimum":2,"multipleOf":2},"flag":{"type":"boolean","enum":[true]}}}}`)
	r := request(doc, "Dto")
	r.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"value == 1 || value == 2 || value == 3 || value == 4", "value >= 2", "value % 2 == 0", "value == true"} {
		if !strings.Contains(source(t, got, "ConsumerDto"), marker) {
			t.Fatal(marker)
		}
	}
	if !strings.Contains(source(t, got, "BridgeDto"), "typealias BridgeDto = kotlin.Int") {
		t.Fatal("invented nominal classifier")
	}
	again, e := core.GenerateWithDiagnostics(r)
	if e != nil || !reflect.DeepEqual(got, again) {
		t.Fatal(e)
	}
}
