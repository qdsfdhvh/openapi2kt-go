package core_test

import (
	"bytes"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzUnionCommonDictionaries(f *testing.F) {
	f.Add(commonDictionaryDocument(`{"additionalProperties":{"type":"string"},"minProperties":1}`, `{"additionalProperties":{"type":"string","minLength":2}}`, "", false))
	f.Add(commonDictionaryDocument(`{"$ref":"#/components/schemas/Mapping"}`, `{"additionalProperties":{"type":"integer"}}`, `,"Mapping":{"additionalProperties":{"type":"string"}}`, true))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 16384 {
			return
		}
		original := bytes.Clone(input)
		rq := core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.fuzz", ModelNameSuffix: "Dto", Validation: true}}
		a, ea := core.GenerateWithDiagnostics(rq)
		b, eb := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(input, original) || (ea == nil) != (eb == nil) || !reflect.DeepEqual(a, b) {
			t.Fatal("input mutation or nondeterministic common dictionary generation")
		}
		if ea != nil && len(a.Artifacts.Files) != 0 {
			t.Fatal("failed generation returned artifacts")
		}
	})
}
