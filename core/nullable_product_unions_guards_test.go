package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableProductUnionsUnsupportedSourcesRemainRejected(t *testing.T) {
	for _, value := range []string{
		`{"nullable":true,"allOf":[{"oneOf":[{"properties":{"id":{"type":"string"}}}]},{"additionalProperties":{"type":"string"}}]}`,
		`{"nullable":true,"allOf":[{"oneOf":[{"properties":{"id":{"type":"string","format":"unsupported"}}}]},{"properties":{"id":{"type":"string"}}}]}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":`+value+`}`), "Dto"))
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("unsupported intersection or discarded source accepted", value, got, err)
		}
	}
}

func TestNullableProductUnionsCountNullInBranchBudget(t *testing.T) {
	for _, count := range []int{239, 240} {
		branches := make([]string, count)
		for i := range branches {
			branches[i] = fmt.Sprintf(`{"properties":{"kind":{"const":"v%d"}}}`, i)
		}
		input := schemas(`{"Value":{"nullable":true,"allOf":[{"oneOf":[` + strings.Join(branches, ",") + `]}]}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 239 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("null carrier bypassed branch cap", count, got, err)
		}
	}
}

func TestNullableProductUnionsSourceDepthAndNullableAliasPolicy(t *testing.T) {
	for _, count := range []int{46, 47} {
		entries := []string{`"Value":` + nullableProductSource(`"nullable":true,`)}
		for i := 0; i < count; i++ {
			target := "Value"
			if i > 0 {
				target = fmt.Sprintf("A%d", i-1)
			}
			entries = append(entries, fmt.Sprintf(`"A%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
		}
		entries = append(entries, fmt.Sprintf(`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/A%d"}},"required":["value"]}`, count-1))
		got, err := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(entries, ",")+`}`), "Dto"))
		if count == 46 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("false source hops reset", count, got, err)
		}
	}
}
