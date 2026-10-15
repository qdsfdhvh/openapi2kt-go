package core_test

import (
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzStructuredConstValues(f *testing.F) {
	for _, literal := range []string{`{}`, `[]`, `{"$ref":"data","type":"data","default":false}`, `[null,true,1e2,"quoted"]`, `{"dup":1,"dup":2}`, `{"n":1e400}`} {
		f.Add(literal, false, false)
	}
	f.Fuzz(func(t *testing.T, literal string, validation, leaf bool) {
		if len(literal) > 4096 {
			t.Skip()
		}
		input := schemas(`{"Value":{"const":` + literal + `}}`)
		if leaf {
			input = fieldDoc(`{"type":"array","items":{"const":` + literal + `}}`)
		}
		r := request(input, "Dto")
		r.Options.Validation = validation
		first, err := core.GenerateWithDiagnostics(r)
		second, again := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(err, again) || err != nil && !reflect.DeepEqual(first, core.GenerationReport{}) {
			t.Fatal("Structured value source drifted or returned partial artifacts", err, again)
		}
	})
}
