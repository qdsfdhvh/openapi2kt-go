package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnionCommonNominalsAliasHeightAndConstructorLimits(t *testing.T) {
	for _, cached := range []bool{false, true} {
		limit := 45
		if cached {
			limit--
		}
		for _, count := range []int{limit, limit + 1} {
			props := []string{}
			for i := 0; i < count; i++ {
				target := "Value"
				if i > 0 {
					target = fmt.Sprintf("Hops/properties/h%d", i-1)
				}
				props = append(props, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
			}
			extra := `,"Value":{"nullable":true,"properties":{"id":{"type":"string"}}},"Hops":{"properties":{` + strings.Join(props, ",") + `}}`
			use := fmt.Sprintf("Hops/properties/h%d", count-1)
			if cached {
				extra += `,"Bridge":{"$ref":"#/components/schemas/` + use + `","nullable":false}`
				use = "Bridge"
			}
			field := `{"$ref":"#/components/schemas/` + use + `","nullable":false}`
			got, err := core.GenerateWithDiagnostics(request(commonDictionaryDocument(field, field, extra, false), "Dto"))
			if count == limit {
				if err != nil {
					t.Fatal("source48", cached, count, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatal("source49", cached, count, got, err)
			}
		}
	}
	for _, width := range []int{239, 240} {
		props := []string{}
		required := []string{}
		for i := 0; i < width; i++ {
			props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Value"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		branch := func(tag string) string {
			return `{"properties":{"kind":{"const":"` + tag + `"},` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]}`
		}
		doc := schemas(`{"Value":{"properties":{"id":{"type":"string"}}},"Choice":{"oneOf":[` + branch("a") + `,` + branch("b") + `]}}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if width == 239 {
			if err != nil {
				t.Fatal("slots240", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("slots241", got, err)
		}
	}
	for _, edges := range []int{16, 17} {
		leaf := `{"type":"string","minLength":1}`
		for i := 0; i < edges; i++ {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		field := `{"$ref":"#/components/schemas/Value"}`
		rq := request(commonDictionaryDocument(field, field, `,"Value":{"properties":{"rows":`+leaf+`},"required":["rows"]}`, false), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if edges == 16 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("nominal erased interior checks", got, err)
		}
	}
}
