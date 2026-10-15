package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnionCommonDictionariesSourceCheckedAndSlotLimits(t *testing.T) {
	for _, depth := range []int{46, 47} {
		leaf := `{"type":"string"}`
		for i := 0; i < depth; i++ {
			leaf = `{"additionalProperties":` + leaf + `}`
		}
		rq := request(commonDictionaryDocument(leaf, leaf, "", false), "Dto")
		got, err := core.GenerateWithDiagnostics(rq)
		if depth == 46 {
			if err != nil {
				t.Fatal("source48", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("source49", got, err)
		}
	}
	for _, depth := range []int{16, 17} {
		leaf := `{"type":"string","minLength":1}`
		for i := 0; i < depth; i++ {
			leaf = `{"additionalProperties":` + leaf + `}`
		}
		rq := request(commonDictionaryDocument(leaf, leaf, "", false), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if depth == 16 {
			if err != nil {
				t.Fatal("checked16", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("checked17", got, err)
		}
	}
	for _, width := range []int{239, 240} {
		props := []string{}
		required := []string{}
		for i := 0; i < width; i++ {
			props = append(props, fmt.Sprintf(`"p%d":{"additionalProperties":{"type":"string"}}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		branch := func(tag string) string {
			return `{"properties":{"kind":{"const":"` + tag + `"},` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]}`
		}
		input := schemas(`{"Choice":{"oneOf":[` + branch("a") + `,` + branch("b") + `]}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if width == 239 {
			if err != nil {
				t.Fatal("slots240", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("slots241", got, err)
		}
	}
}

func TestUnionCommonDictionariesAliasCyclesRemainErrors(t *testing.T) {
	field := `{"additionalProperties":{"$ref":"#/components/schemas/Mapping"}}`
	got, err := core.GenerateWithDiagnostics(request(commonDictionaryDocument(field, field, `,"Mapping":`+field, false), "Dto"))
	if err == nil || err.(core.Diagnostics)[0].Code != "E_REF_CYCLE" || len(got.Artifacts.Files) != 0 {
		t.Fatal(got, err)
	}
}
