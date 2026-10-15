package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumBoundsRetainVocabularySlotsAndSourceBudgets(t *testing.T) {
	for _, kind := range []string{"int", "long", "vocabulary", "sourceDepth", "collections"} {
		limit := 240
		if kind == "long" {
			limit = 120
		}
		if kind == "sourceDepth" {
			limit = 47
		}
		if kind == "collections" {
			limit = 16
		}
		for _, count := range []int{limit, limit + 1} {
			properties := []string{}
			required := []string{}
			switch kind {
			case "vocabulary":
				values := make([]string, count)
				for i := range values {
					values[i] = fmt.Sprint(i)
				}
				properties = append(properties, `"value":{"type":"integer","enum":[`+strings.Join(values, ",")+`],"minimum":1,"multipleOf":2}`)
			case "sourceDepth", "collections":
				schema := `{"type":"integer","enum":[1,2],"minimum":2}`
				for i := range count {
					if kind == "collections" {
						schema = `{"type":"array","items":` + schema + `}`
					} else {
						schema = fmt.Sprintf(`{"properties":{"n%d":%s},"required":["n%d"]}`, i, schema, i)
					}
				}
				properties = append(properties, `"value":`+schema)
			default:
				for i := range count {
					format := ""
					if kind == "long" {
						format = `,"format":"int64"`
						required = append(required, fmt.Sprintf(`"n%d"`, i))
					}
					properties = append(properties, fmt.Sprintf(`"n%d":{"type":"integer"%s,"enum":[1,2],"minimum":2}`, i, format))
				}
			}
			r := request(schemas(`{"Record":{"properties":{`+strings.Join(properties, ",")+`},"required":[`+strings.Join(required, ",")+`]}}`), "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			if (err == nil) != (count == limit) {
				t.Fatal(kind, count, err)
			}
			if err != nil {
				code := "E_MODEL_LIMIT"
				if kind == "sourceDepth" || kind == "collections" {
					code = "E_SCHEMA_DEPTH"
				}
				if err.(core.Diagnostics)[0].Code != code || len(got.Artifacts.Files) != 0 {
					t.Fatal(kind, err)
				}
			}
		}
	}
}
