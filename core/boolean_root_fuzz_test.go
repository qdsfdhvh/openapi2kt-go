package core_test

import (
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzGenerateBooleanRoot(f *testing.F) {
	for _, seed := range []string{
		`{"openapi":"3.1.0","components":{"schemas":{"Any":true}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/Any"},"Any":true}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"Any":false}}}`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 100000 {
			t.Skip()
		}
		r := request(input, "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		again, e := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(err, e) {
			t.Fatal("nondeterministic")
		}
		if err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("partial failure")
		}
	})
}
