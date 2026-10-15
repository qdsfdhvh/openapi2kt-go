package core_test

import (
	"bytes"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNullableObjectReferenceMembers(f *testing.F) {
	f.Add(schemas(`{"Value":{"nullable":true,"properties":{"kind":{"const":"a"},"id":{"type":"string"}}},"Choice":{"oneOf":[{"$ref":"#/components/schemas/Value"},{"properties":{"kind":{"const":"b"}}}]}}`))
	f.Add(schemas(`{"Value":{"properties":{"kind":{"const":"a"},"id":{"type":"string"}}},"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Choice":{"oneOf":[{"$ref":"#/components/schemas/Alias"},{"properties":{"kind":{"const":"b"}}}]}}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 16384 {
			return
		}
		original := bytes.Clone(input)
		rq := core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.fuzz", ModelNameSuffix: "Dto", Validation: true}}
		a, ea := core.GenerateWithDiagnostics(rq)
		b, eb := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(input, original) || (ea == nil) != (eb == nil) || !reflect.DeepEqual(a, b) {
			t.Fatal("input mutation or nondeterministic nullable product generation")
		}
		if ea != nil && len(a.Artifacts.Files) != 0 {
			t.Fatal("failed generation returned artifacts")
		}
	})
}
