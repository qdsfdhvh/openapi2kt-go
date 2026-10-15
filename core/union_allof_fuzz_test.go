package core_test

import (
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzGenerateUnionAllOf(f *testing.F) {
	input, err := os.ReadFile("../tests/fixtures/http/union-allof.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(input)
	f.Add(schemas(`{"Root":{"allOf":[{"$ref":"#/components/schemas/Root"},{"oneOf":[{"type":"object"}]}]}}`))
	f.Add(schemas(`{"Root":{"allOf":[{"oneOf":[{"type":"object"}]},{"type":"object"}]}}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 100000 {
			t.Skip()
		}
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			a, e := core.GenerateWithDiagnostics(r)
			b, e2 := core.GenerateWithDiagnostics(r)
			if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(e, e2) {
				t.Fatal("nondeterministic")
			}
			if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial failed report")
			}
		}
	})
}
