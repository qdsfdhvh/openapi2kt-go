package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInternalNativeEnumPointersRetainJVMAndVocabularyLimits(t *testing.T) {
	for _, kind := range []string{"fields", "longs", "vocabulary"} {
		limit := 240
		if kind == "longs" {
			limit = 120
		}
		for _, count := range []int{limit, limit + 1} {
			leaf := `{"type":"integer","enum":[1,2],"minimum":2}`
			if kind == "longs" {
				leaf = `{"type":"integer","format":"int64","enum":[1,2],"minimum":2}`
			}
			properties := []string{}
			required := []string{}
			if kind == "vocabulary" {
				values := make([]string, count)
				for i := range values {
					values[i] = fmt.Sprint(i)
				}
				leaf = `{"type":"integer","enum":[` + strings.Join(values, ",") + `],"minimum":1}`
				properties = append(properties, `"value":{"$ref":"#/components/schemas/Bridge"}`)
			} else {
				for i := range count {
					properties = append(properties, fmt.Sprintf(`"n%d":{"$ref":"#/components/schemas/Bridge"}`, i))
					if kind == "longs" {
						required = append(required, fmt.Sprintf(`"n%d"`, i))
					}
				}
			}
			r := request(schemas(`{"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf"},"Source":{"properties":{"leaf":`+leaf+`}},"Consumer":{"properties":{`+strings.Join(properties, ",")+`},"required":[`+strings.Join(required, ",")+`]}}`), "Dto")
			r.Options.Validation = true
			got, e := core.GenerateWithDiagnostics(r)
			if (e == nil) != (count == limit) {
				t.Fatal(kind, count, e)
			}
			if e != nil && (e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
				t.Fatal(e)
			}
		}
	}
}
