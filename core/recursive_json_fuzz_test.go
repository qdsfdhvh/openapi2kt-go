package core_test

import (
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzGenerateRecursiveJSON(f *testing.F) {
	input, err := os.ReadFile("../tests/fixtures/http/recursive-json.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(input)
	mutual, err := os.ReadFile("../tests/fixtures/http/mutual-recursive-json.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(mutual)
	graph, err := os.ReadFile("../tests/fixtures/http/recursive-json-reference-graph.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(graph)
	f.Add([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"anyOf":[{"$ref":"#/components/schemas/Value"}]}}}}`))
	f.Add([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"type":"object","properties":{"child":{"$ref":"#/components/schemas/Value"}}}}}}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 100000 {
			t.Skip()
		}
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			a, e := core.GenerateWithDiagnostics(r)
			b, e2 := core.GenerateWithDiagnostics(r)
			if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(e, e2) {
				t.Fatal("determinism")
			}
			if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial report")
			}
		}
	})
}
