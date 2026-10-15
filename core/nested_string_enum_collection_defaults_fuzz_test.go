package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzNestedStringEnumCollectionDefaultOriginalValues(f *testing.F) {
	for _, seed := range []string{`[{"a":"one"}]`, `[{"/~":null}]`, `[{"z":"future","a":false}]`, `[]`, `null`, `[[]]`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, annotation string) {
		if len(annotation) > 4096 || !json.Valid([]byte(annotation)) {
			t.Skip()
		}
		input := schemas(`{"Holder":{"properties":{"value":{"type":"array","nullable":true,"items":{"additionalProperties":{"enum":[null,"one","two"],"nullable":true},"nullable":true},"default":` + annotation + `}}}}`)
		var first core.GenerationReport
		var firstError string
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				r := request(input, "Dto")
				r.Options.ModelStyle = style
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					if !reflect.DeepEqual(got, core.GenerationReport{}) {
						t.Fatal("partial failure")
					}
					if firstError != "" && firstError != err.Error() {
						t.Fatal("mode-dependent membership", firstError, err)
					}
					firstError = err.Error()
				} else {
					if firstError != "" {
						t.Fatal("mode-dependent acceptance")
					}
					repeated, e := core.GenerateWithDiagnostics(r)
					if e != nil || !reflect.DeepEqual(got, repeated) {
						t.Fatal("nondeterministic", e)
					}
					if style == core.DataClass {
						if !reflect.DeepEqual(first, core.GenerationReport{}) && !reflect.DeepEqual(first, got) {
							t.Fatal("validation closed open vocabulary")
						}
						first = got
					}
				}
			}
		}
	})
}
