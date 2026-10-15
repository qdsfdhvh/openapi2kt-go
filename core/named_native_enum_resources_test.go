package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNamedNativeEnumsRetainOriginalSlotsAndMemberLimits(t *testing.T) {
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
			props, required := []string{}, []string{}
			if kind == "vocabulary" {
				values := make([]string, count)
				for i := range values {
					values[i] = fmt.Sprint(i)
				}
				leaf = `{"type":"integer","enum":[` + strings.Join(values, ",") + `],"minimum":1}`
				props = append(props, `"value":{"$ref":"#/components/schemas/Bridge"}`)
			} else {
				for i := range count {
					props = append(props, fmt.Sprintf(`"n%d":{"$ref":"#/components/schemas/Bridge"}`, i))
					if kind == "longs" {
						required = append(required, fmt.Sprintf(`"n%d"`, i))
					}
				}
			}
			r := request(schemas(`{"Code":`+leaf+`,"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"},"Source":{"properties":{"hop":{"$ref":"#/components/schemas/Code"}}},"Consumer":{"properties":{`+strings.Join(props, ",")+`},"required":[`+strings.Join(required, ",")+`]}}`), "Dto")
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
