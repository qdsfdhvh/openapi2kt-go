package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzDiscardedStructuredConst(f *testing.F) {
	for _, literal := range []string{`{}`, `[]`, `{"$ref":"literal data","type":"data","enum":[null]}`, `[null,{"n":1e400}]`, `{"same":1,"same":2}`} {
		f.Add(literal, false, false)
	}
	f.Fuzz(func(t *testing.T, literal string, extra, checked bool) {
		if len(literal) > 4096 {
			t.Skip()
		}
		body := `{"Choice":{"oneOf":[{"allOf":[{"properties":{"payload":{"const":` + literal + `}}},{"properties":{"payload":{"type":"string"},"tag":{"const":"one"}}}]},{"properties":{"tag":{"const":"two"}}}]}}`
		owner := "ChoiceDto"
		if extra {
			body = `{"Holder":{"properties":{"value":{"type":"string"}},"additionalProperties":{"const":` + literal + `}}}`
			owner = "HolderDto"
		}
		r := request(schemas(body), "Dto")
		r.Options.Validation = checked
		first, e := core.GenerateWithDiagnostics(r)
		again, e2 := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(first, again) || !reflect.DeepEqual(e, e2) || e != nil && !reflect.DeepEqual(first, core.GenerationReport{}) {
			t.Fatal("Discarded-source result drift or partial files", e, e2)
		}
		if e == nil && json.Valid([]byte(literal)) {
			code := source(t, first, owner)
			if strings.Contains(code, "openapi-kt object const") || strings.Contains(code, "openapi-kt array const") {
				t.Fatal("Discarded literal acquired Kotlin state", code)
			}
			for _, w := range first.Warnings {
				if w.Code == "W_CONST_STRUCTURED_APPROXIMATION" {
					t.Fatal("Discarded literal acquired typed value/body warning", w)
				}
			}
		}
	})
}
