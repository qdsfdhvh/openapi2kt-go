package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNamedNativeEnumsRetainChecksThroughInternalChainsAndBorrowedCollections(t *testing.T) {
	doc := schemas(`{"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"},"BorrowedRows":{"$ref":"#/components/schemas/Source/properties/rows"},"Source":{"properties":{"hop":{"$ref":"#/components/schemas/Alias"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"Alias":{"$ref":"#/components/schemas/Code"},"Code":{"type":"integer","enum":[1,2,3,4],"minimum":2,"multipleOf":2},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Bridge"},"rows":{"$ref":"#/components/schemas/BorrowedRows"}}}}`)
	r := request(doc, "Dto")
	r.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"value == 1 || value == 2 || value == 3 || value == 4", "value >= 2", "value % 2 == 0"} {
		if !strings.Contains(source(t, got, "ConsumerDto"), marker) {
			t.Fatal(marker)
		}
	}
	if !strings.Contains(source(t, got, "BridgeDto"), "typealias BridgeDto = kotlin.Int") {
		t.Fatal("nominal enum bridge")
	}
	again, e := core.GenerateWithDiagnostics(r)
	if e != nil || !reflect.DeepEqual(got, again) {
		t.Fatal(e)
	}
}
