package core_test

import (
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzGenerateBooleanHTTP(f *testing.F) {
	for _, input := range []string{
		`{"openapi":"3.1.0","paths":{"/x":{"post":{"requestBody":{"content":{"application/json":{"schema":true}}},"responses":{"200":{"content":{"application/json":{"schema":true}}}}}}}}`,
		`{"openapi":"3.1.0","paths":{"/x":{"get":{"responses":{"200":{"content":{"application/json":{"schema":true}}},"201":{"content":{"application/json":{"schema":false}}}}}}}}`,
		`{"openapi":"3.0.3","paths":{"/x":{"get":{"responses":{"200":{"content":{"application/json":{"schema":true}}}}}}}}`,
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
			t.Fatal("partial report")
		}
	})
}
