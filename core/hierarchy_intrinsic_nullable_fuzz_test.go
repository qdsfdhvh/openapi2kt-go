package core_test

import (
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzGenerateIntrinsicNullableHierarchy(f *testing.F) {
	input, err := os.ReadFile("../tests/fixtures/http/hierarchy-intrinsic-nullable.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(input)
	f.Add(schemas(`{"Root":{"nullable":true,"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"nullable":true,"properties":{"id":{"type":"string"}}}}`))
	f.Add(schemas(`{"Root":{"nullable":true,"oneOf":[{"$ref":"#/components/schemas/Root"}],"discriminator":{"propertyName":"kind"}}}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 100000 {
			t.Skip()
		}
		for _, checked := range []bool{false, true} {
			req := request(input, "Dto")
			req.Options.Validation = checked
			a, err := core.GenerateWithDiagnostics(req)
			b, err2 := core.GenerateWithDiagnostics(req)
			if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(err, err2) {
				t.Fatal("nondeterministic")
			}
			if err != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial failed report")
			}
		}
	})
}
