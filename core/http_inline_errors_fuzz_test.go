package core_test

import (
	"bytes"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzHTTPInlineErrors(f *testing.F) {
	f.Add(httpDoc(`{"responses":{"204":{},"400":{"content":{"application/json":{"schema":{"properties":{"message":{"type":"string"}}}}}}}}`))
	f.Add(httpDoc(`{"responses":{"default":{"content":{"application/json":{"schema":{"type":["object","null"],"properties":{"message":{"type":"string"}}}}}}}}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 16384 {
			return
		}
		original := bytes.Clone(input)
		rq := core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.fuzz", ModelNameSuffix: "Dto", Validation: true}}
		a, ea := core.GenerateWithDiagnostics(rq)
		b, eb := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(input, original) || (ea == nil) != (eb == nil) || !reflect.DeepEqual(a, b) {
			t.Fatal("input mutation or nondeterministic default generation")
		}
		if ea != nil && len(a.Artifacts.Files) != 0 {
			t.Fatal("failed generation returned artifacts")
		}
	})
}
