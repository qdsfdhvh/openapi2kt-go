package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableAllOfDTOExplicitRootPhysicalAndTypedViews(t *testing.T) {
	for _, intent := range []string{`"nullable":true`, `"type":["object","null"]`} {
		input := schemas(`{"Base":{"properties":{"id":{"type":"string"}},"required":["id"]},"Value":{` + intent + `,"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"child":{"$ref":"#/components/schemas/Value"}}}]},"Alias":{"$ref":"#/components/schemas/Value"},"False":{"$ref":"#/components/schemas/Alias","nullable":false},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Value"},"narrow":{"$ref":"#/components/schemas/False"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}},"required":["value","narrow","rows"]}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(intent, checked, err)
			}
			physical := source(t, got, "ValueDto")
			if !strings.Contains(physical, "public data class ValueDto(") || strings.Contains(physical, "typealias") {
				t.Fatal(physical)
			}
			if !strings.Contains(source(t, got, "AliasDto"), "= org.example.models.models.ValueDto?\n") {
				t.Fatal("inherited nullable alias")
			}
			if !strings.Contains(source(t, got, "FalseDto"), "= org.example.models.models.ValueDto\n") {
				t.Fatal("false alias")
			}
			for _, part := range []string{"val value: org.example.models.models.ValueDto? = null", "val narrow: org.example.models.models.ValueDto,", "List<org.example.models.models.ValueDto?>"} {
				if !strings.Contains(source(t, got, "HolderDto"), part) {
					t.Fatal(part, source(t, got, "HolderDto"))
				}
			}
		}
	}
}
