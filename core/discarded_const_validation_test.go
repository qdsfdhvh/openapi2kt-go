package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDiscardedConstValidationSharedAndUniqueSourcesKeepProjectionArtifacts(t *testing.T) {
	for _, role := range []string{"shared", "unique", "overridden"} {
		literal := `{"payload":"` + strings.Repeat("a", 32768) + `","$ref":"literal data"}`
		build := func(retain bool) []byte {
			nodes := []string{}
			ap := `true`
			if retain {
				ap = `{"const":` + literal + `}`
			}
			if role == "shared" {
				nodes = append(nodes, `"Base":{"properties":{"value":{"type":"string"}},"additionalProperties":`+ap+`}`)
			}
			for i := range 32 {
				body := `{"properties":{"value":{"type":"string"}},"additionalProperties":` + ap + `}`
				if role == "shared" {
					body = `{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"owned":{"type":"string"}}}]}`
				}
				if role == "overridden" {
					first := `{"type":"string"}`
					if retain {
						first = `{"const":` + literal + `}`
					}
					body = `{"allOf":[{"properties":{"value":` + first + `}},{"properties":{"value":{"type":"string"}}}]}`
				}
				nodes = append(nodes, fmt.Sprintf(`"Owner%d":%s`, i, body))
			}
			return schemas(`{` + strings.Join(nodes, ",") + `}`)
		}
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				r := request(build(true), "Dto")
				r.Options.ModelStyle = style
				r.Options.Validation = checked
				got, e := core.GenerateWithDiagnostics(r)
				if e != nil {
					t.Fatal(role, style, checked, e)
				}
				r.Document = build(false)
				control, e := core.GenerateWithDiagnostics(r)
				if e != nil || !reflect.DeepEqual(got, control) {
					t.Fatal("Ignored expression optimization changed artifacts or original warnings", role, style, checked, e)
				}
			}
		}
	}
}
