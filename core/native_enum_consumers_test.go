package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumTypedConsumersRetainMembershipAndNullViews(t *testing.T) {
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","enum":[1,2]},"Alias":{"$ref":"#/components/schemas/Code"},"Record":{"properties":{"code":{"$ref":"#/components/schemas/Alias"},"rows":{"type":"array","items":{"type":["integer","null"],"enum":[1,2]}},"map":{"type":"object","additionalProperties":{"type":"boolean","enum":[true]}}}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"paths":{"/read":{"get":{"operationId":"readCode","parameters":[{"name":"code","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Alias"}}],"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Alias"}}}}}}}}}`)
	r := request(doc, "Dto")
	r.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"RecordDto", "DefaultApi"} {
		if !strings.Contains(source(t, got, name), "does not belong to enum.") {
			t.Fatal("missing typed enum consumer checks", name)
		}
	}
	if !strings.Contains(source(t, got, "CodeDto"), "typealias CodeDto = kotlin.Int") || strings.Contains(source(t, got, "CodeDto"), "require(") {
		t.Fatal("alias became wrapper")
	}
	if !strings.Contains(source(t, got, "RecordDto"), "value == true") {
		t.Fatal("map leaf check missing")
	}
	for _, warning := range got.Warnings {
		if warning.Code == "W_ENUM_NATIVE_APPROXIMATION" {
			t.Fatal("enforced source still approximate", warning)
		}
	}
}
