package core_test

import (
	"bytes"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzBorrowedCollectionProductUnionLeaves(f *testing.F) {
	f.Add([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Node":{"allOf":[{"oneOf":[{"properties":{"kind":{"const":"id"},"id":{"type":"string"}}},{"properties":{"kind":{"const":"name"},"name":{"type":"string"}}}]},{"properties":{"common":{"type":"string"}}}]},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}}}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 16384 {
			return
		}
		original := bytes.Clone(input)
		rq := core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.fuzz", ModelNameSuffix: "Dto", Validation: true}}
		a, errA := core.GenerateWithDiagnostics(rq)
		b, errB := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(input, original) || (errA == nil) != (errB == nil) || !reflect.DeepEqual(a, b) {
			t.Fatal("input mutation or nondeterministic product generation")
		}
		if errA != nil && len(a.Artifacts.Files) != 0 {
			t.Fatal("failed product generation returned artifacts")
		}
	})
}
