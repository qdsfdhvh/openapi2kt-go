package core_test

import (
	"bytes"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNullableProductUnions(f *testing.F) {
	f.Add(schemas(`{"Product":` + nullableProductSource(`"nullable":true,`) + `}`))
	f.Add(fieldDoc(nullableProductSource(`"type":["object","null"],`)))
	f.Add(schemas(`{"Product":` + intersectionProduct(true, true) + `,"Alias":{"$ref":"#/components/schemas/Product","nullable":false}}`))
	f.Add(fieldDoc(intersectionProduct(true, false)))
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
