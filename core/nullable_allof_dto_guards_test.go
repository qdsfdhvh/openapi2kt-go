package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableAllOfDTOFinalConstructorAndShapeGuards(t *testing.T) {
	// Original nullable product input is now an actual union with a Null
	// carrier, rather than an ordinary nullable DTO or copied shape.
	original := `{"nullable":true,"allOf":[{"oneOf":[{"properties":{"kind":{"const":"a"}}},{"properties":{"kind":{"const":"b"}}}]}]}`
	accepted, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":`+original+`}`), "Dto"))
	if err != nil || !strings.Contains(source(t, accepted, "ValueDto"), "NullValueDto") {
		t.Fatal("original nullable product did not retain union carrier", err)
	}
	for _, body := range []string{
		`{"nullable":true,"allOf":[{"type":"object"}]}`,
		`{"nullable":true,"allOf":[{"properties":{"pin":{"const":"fixed"}}}]}`,
		`{"nullable":true,"allOf":[{"additionalProperties":{"type":"string"}}]}`,
		`{"allOf":[{"nullable":true,"properties":{"id":{"type":"string"}}}]}`,
		`{"nullable":true,"allOf":[{"properties":{"id":{"type":"string"}}}],"minProperties":1}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Value":`+body+`}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal(body, checked, got, err)
			}
		}
	}
	value := `{"nullable":true,"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]}]}`
	for _, tail := range []string{
		`,"Other":{"allOf":[{"$ref":"#/components/schemas/Value","nullable":false},{"properties":{"more":{"type":"string"}}}]}`,
		`,"Other":{"oneOf":[{"$ref":"#/components/schemas/False"},{"properties":{"ok":{"type":"boolean"}}}]}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":`+value+`,"False":{"$ref":"#/components/schemas/Value","nullable":false}`+tail+`}`), "Dto"))
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("nullable nominal shape erased", tail, got, err)
		}
	}
	// Original discarded fields remain checked even if the last member wins.
	input := schemas(`{"Value":{"nullable":true,"allOf":[{"properties":{"id":{"type":"string","format":"unsupported"}}},{"properties":{"id":{"type":"string"}}}]}}`)
	failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Value/allOf/0/properties/id/format")
}

func TestNullableAllOfDTOPrimitiveChecksAndIndependentLimits(t *testing.T) {
	input := schemas(`{"Value":{"nullable":true,"allOf":[{"properties":{"id":{"type":"string","minLength":2}},"required":["id"]},{"properties":{"more":{"type":"string"}}}]},"Alias":{"$ref":"#/components/schemas/Value"}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if checked {
			if err != nil || !strings.Contains(source(t, got, "ValueDto"), "require(length >= 2)") {
				t.Fatal(got, err)
			}
		} else if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("validation-off widened bounds", got, err)
		}
	}
	for _, depth := range []int{16, 17} {
		leaf := `{"type":"string","minLength":1}`
		for i := 0; i < depth; i++ {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		rq := request(schemas(`{"Value":{"nullable":true,"allOf":[{"properties":{"rows":`+leaf+`},"required":["rows"]}]}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if depth == 16 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal(got, err)
		}
	}
	for _, width := range []int{240, 241} {
		props := []string{}
		required := []string{}
		for i := 0; i < width; i++ {
			props = append(props, fmt.Sprintf(`"p%d":{"type":"boolean"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":{"nullable":true,"allOf":[{"properties":{`+strings.Join(props, ",")+`},"required":[`+strings.Join(required, ",")+`]}]}}`), "Dto"))
		if width == 240 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal(got, err)
		}
	}
	for _, count := range []int{46, 47} {
		target := `{"nullable":true,"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]}]}`
		rq := request(rootSpecialChain(count, target, fmt.Sprintf(`{"properties":{"value":{"$ref":"#/components/schemas/A%02d","nullable":false}},"required":["value"]}`, count-1)), "Dto")
		got, err := core.GenerateWithDiagnostics(rq)
		if count == 46 {
			if err != nil {
				t.Fatal("48source", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("49source", got, err)
		}
	}
}
