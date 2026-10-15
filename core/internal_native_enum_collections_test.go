package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInternalNativeEnumPointerForwardRootAndBorrowedCollections(t *testing.T) {
	doc := schemas(`{"ARows":{"$ref":"#/components/schemas/Source/properties/rows"},"BMap":{"$ref":"#/components/schemas/Source/properties/map"},"ZBridge":{"$ref":"#/components/schemas/Source/properties/code"},"Consumer":{"properties":{"rows":{"$ref":"#/components/schemas/ARows"},"map":{"$ref":"#/components/schemas/BMap"}}},"Source":{"properties":{"code":{"type":"integer","enum":[1,2,3,4],"minimum":2},"rows":{"type":"array","items":{"$ref":"#/components/schemas/ZBridge"}},"map":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Source/properties/code"}}}}}`)
	r := request(doc, "Dto")
	r.Options.Validation = true
	got, e := core.GenerateWithDiagnostics(r)
	if e != nil {
		t.Fatal(e)
	}
	text := source(t, got, "ConsumerDto")
	if !strings.Contains(text, "value.values.forEach") || !strings.Contains(text, "value >= 2") {
		t.Fatal(text)
	}
}
