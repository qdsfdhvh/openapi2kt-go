package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnionCommonDictionariesIncomingAliasWorkAndCachedFalse(t *testing.T) {
	for _, cached := range []bool{false, true} {
		limit := 44
		if cached {
			limit--
		}
		for _, count := range []int{limit, limit + 1} {
			props := []string{}
			for i := 0; i < count; i++ {
				target := "Mapping"
				if i > 0 {
					target = fmt.Sprintf("Hops/properties/h%d", i-1)
				}
				props = append(props, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
			}
			extra := `,"Mapping":{"additionalProperties":{"type":"string"}},"Hops":{"properties":{` + strings.Join(props, ",") + `}}`
			use := fmt.Sprintf("Hops/properties/h%d", count-1)
			if cached {
				extra += `,"Bridge":{"$ref":"#/components/schemas/` + use + `","nullable":false}`
				use = "Bridge"
			}
			field := `{"$ref":"#/components/schemas/` + use + `","nullable":false}`
			got, err := core.GenerateWithDiagnostics(request(commonDictionaryDocument(field, field, extra, false), "Dto"))
			if count == limit {
				if err != nil {
					t.Fatal("source48 cached", cached, count, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatal("source49 cached", cached, count, got, err)
			}
		}
	}
}

func TestUnionCommonDictionariesLateInlineHTTPPlan(t *testing.T) {
	value := `{"additionalProperties":{"$ref":"#/components/schemas/Record"}}`
	union := `{"oneOf":[{"properties":{"kind":{"const":"a"},"value":` + value + `},"required":["value"]},{"properties":{"kind":{"const":"b"},"value":` + value + `},"required":["value"]}]}`
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"id":{"type":"string"}},"required":["id"]}}},"paths":{"/submit":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":` + union + `}}},"responses":{"204":{}}}}}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("first late common dictionary", checked, err)
		}
		text := ""
		for _, file := range got.Artifacts.Files {
			text += file.Content
		}
		if !strings.Contains(text, "abstract val value: kotlin.collections.Map<kotlin.String, org.example.models.models.RecordDto>") {
			t.Fatal(text)
		}
	}
}
