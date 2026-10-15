package core_test

import (
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzStructuredConstFields(f *testing.F) {
	for _, literal := range []string{`{}`, `[]`, `{"type":"data","$ref":"data","enum":[null]}`, `[null,true,1e2,"quoted"]`, `{"duplicate":1,"duplicate":2}`, `{"n":1e400}`} {
		f.Add(literal, false, false)
	}
	f.Fuzz(func(t *testing.T, literal string, validation, plain bool) {
		if len(literal) > 4096 {
			t.Skip()
		}
		r := request(fieldDoc(`{"const":`+literal+`}`), "Dto")
		r.Options.Validation = validation
		if plain {
			r.Options.ModelStyle = core.PlainClass
		}
		first, err := core.GenerateWithDiagnostics(r)
		second, again := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(err, again) || err != nil && !reflect.DeepEqual(first, core.GenerationReport{}) {
			t.Fatal("Structured const literal generation drifted or leaked a partial report", err, again)
		}
	})
}
