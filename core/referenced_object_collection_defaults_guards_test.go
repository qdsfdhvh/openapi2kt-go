package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedObjectCollectionDefaultsGuards(t *testing.T) {
	for name, input := range map[string]string{
		"element missing required":            `{"Item":{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[{}]}}}}`,
		"element unknown key":                 `{"Item":{"type":"object","properties":{"name":{"type":"string"}}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[{"zz":1}]}}}}`,
		"target own default":                  `{"Item":{"type":"object","properties":{"name":{"type":"string"}},"default":{"name":"t"}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[{"name":"x"}]}}}}`,
		"named collection dto target default": `{"Item":{"type":"object","properties":{"name":{"type":"string"}},"default":{"name":"t"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":[{"name":"x"}]}}}}`,
		"dict element wrong kind":             `{"Item":{"type":"object","properties":{"name":{"type":"string"}}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":{"name":"x"}}}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			for _, checked := range []bool{false, true} {
				r := request(schemas(input), "Dto")
				r.Options.Validation = checked
				report, err := core.GenerateWithDiagnostics(r)
				if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatalf("excluded referenced collection role leaked (checked=%v): %v", checked, err)
				}
			}
		})
	}
	// Scalar/enum leaves keep their existing referenced collection roles.
	for _, input := range []string{
		`{"Rows":{"type":"array","items":{"type":"integer","format":"int64"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":[1,2]}}}}`,
		`{"Item":{"type":"string","enum":["a"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":["a"]}}}}`,
	} {
		r := request(schemas(input), "Dto")
		r.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("scalar/enum referenced collection regressed", input, err)
		}
	}
}

func TestReferencedObjectCollectionDefaultsBudgets(t *testing.T) {
	big := strings.Repeat("a", 33000)
	large := schemas(`{"Item":{"type":"object","properties":{"s":{"type":"string"}}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[{"s":"` + big + `"}]}}}}`)
	if _, err := core.GenerateWithDiagnostics(request(large, "Dto")); err == nil {
		t.Fatal("oversized referenced object collection default accepted")
	}
}
