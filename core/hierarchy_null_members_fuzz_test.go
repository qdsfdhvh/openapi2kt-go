package core_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzHierarchyNullableMembers(f *testing.F) {
	b, e := os.ReadFile("../tests/fixtures/http/hierarchy-null-members.json")
	if e != nil {
		f.Fatal(e)
	}
	f.Add(b)
	f.Add(schemas(`{"Root":{"oneOf":[{"type":"null"},{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag"}},"Leaf":{"type":"object"}}`))
	f.Add(schemas(`{"Root":{"oneOf":[{"type":"null"},{"$ref":"#/components/schemas/Root"}],"discriminator":{"propertyName":"tag"}}}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 100000 {
			t.Skip()
		}
		for _, validation := range []bool{false, true} {
			req := request(b, "Dto")
			req.Options.Validation = validation
			a, e := core.GenerateWithDiagnostics(req)
			c, e2 := core.GenerateWithDiagnostics(req)
			if !reflect.DeepEqual(a, c) || fmt.Sprint(e) != fmt.Sprint(e2) {
				t.Fatal("nondeterministic hierarchy members")
			}
			if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial failed plan")
			}
		}
	})
}
