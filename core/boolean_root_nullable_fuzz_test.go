package core_test

import (
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzGenerateBooleanRootNullable(f *testing.F) {
	for _, input := range []string{
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":true}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/B","nullable":false},"B":{"$ref":"#/components/schemas/Z","nullable":true},"Z":true}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":false}}}`,
	} {
		f.Add([]byte(input))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 100000 {
			t.Skip()
		}
		r := request(input, "Dto")
		a, e := core.GenerateWithDiagnostics(r)
		b, e2 := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(e, e2) {
			t.Fatal("determinism")
		}
		if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
			t.Fatal("partial", a, e)
		}
	})
}
