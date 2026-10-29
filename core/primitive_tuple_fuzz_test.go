package core_test

import (
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzGeneratePrimitiveTuple(f *testing.F) {
	for _, seed := range []string{
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"array","prefixItems":[{"enum":[null,"ready"]}],"default":[null,"ready"]}}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"array","prefixItems":[{"type":"integer","enum":[1,2]},{"enum":["ready"]}],"items":true,"default":[null,{"enum":"data"}]}}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}]}}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"array","prefixItems":[{"type":"string","minimum":0}],"items":{"type":"boolean"}}}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"array","prefixItems":[{"type":["string","null"]}],"items":false}}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"array","prefixItems":[{"type":"string"}],"items":true}}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"array","prefixItems":[{"type":"integer"}],"items":false,"default":[]}}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"array","prefixItems":[{"type":"string","format":"date"},{"type":"string","format":"date-time"}]}}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"array","prefixItems":[{"type":"string","format":"date-time","nullable":true}],"items":{"type":"string","format":"date","nullable":true}}}}}`,
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
