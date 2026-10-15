package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumValidationRetainsSourceAndJVMSlotBudgets(t *testing.T) {
	for _, kind := range []string{"int", "long", "vocabulary", "sourceDepth"} {
		limit := 240
		if kind == "long" {
			limit = 120
		}
		if kind == "sourceDepth" {
			limit = 47
		}
		for _, count := range []int{limit, limit + 1} {
			properties := []string{}
			switch kind {
			case "vocabulary":
				values := make([]string, count)
				for i := range values {
					values[i] = fmt.Sprint(i)
				}
				properties = append(properties, `"value":{"type":"integer","enum":[`+strings.Join(values, ",")+`]}`)
			case "sourceDepth":
				schema := `{"type":"integer","enum":[1,2]}`
				for i := range count {
					schema = fmt.Sprintf(`{"properties":{"n%d":%s},"required":["n%d"]}`, i, schema, i)
				}
				properties = append(properties, `"value":`+schema)
			default:
				for i := range count {
					format := ""
					if kind == "long" {
						format = `,"format":"int64"`
					}
					properties = append(properties, fmt.Sprintf(`"n%d":{"type":"integer"%s,"enum":[1,2]}`, i, format))
				}
			}
			// Required Long uses two JVM serialized slots; optional boxing uses one.
			required := []string{}
			if kind == "long" {
				for i := range count {
					required = append(required, fmt.Sprintf(`"n%d"`, i))
				}
			}
			doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{` + strings.Join(properties, ",") + `},"required":[` + strings.Join(required, ",") + `]}}}}`)
			r := request(doc, "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			if (err == nil) != (count == limit) {
				t.Fatal(kind, count, err)
			}
			if err != nil {
				want := "E_MODEL_LIMIT"
				if kind == "sourceDepth" {
					want = "E_SCHEMA_DEPTH"
				}
				if len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != want {
					t.Fatal(kind, err)
				}
			}
		}
	}
}
