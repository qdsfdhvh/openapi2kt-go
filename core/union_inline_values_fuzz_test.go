package core_test

import (
	"bytes"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzUnionInlineValues(f *testing.F) {
	for _, field := range []string{`{"properties":{"id":{"type":"string"}}}`, `{"type":"array","items":{"properties":{"id":{"type":"string"}}}}`, `{"additionalProperties":{"properties":{"id":{"type":"string"}}}}`} {
		f.Add(commonDictionaryDocument(field, field, "", false))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 16384 {
			return
		}
		original := bytes.Clone(input)
		rq := core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.fuzz", ModelNameSuffix: "Dto", Validation: true}}
		a, ea := core.GenerateWithDiagnostics(rq)
		b, eb := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(input, original) || (ea == nil) != (eb == nil) || !reflect.DeepEqual(a, b) {
			t.Fatal("input mutation or nondeterministic inline union generation")
		}
		if ea != nil && len(a.Artifacts.Files) != 0 {
			t.Fatal("failed generation returned artifacts")
		}
	})
}
