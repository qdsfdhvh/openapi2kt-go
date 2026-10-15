package core_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzHierarchyNestedOwners(f *testing.F) {
	b, e := os.ReadFile("../tests/fixtures/http/hierarchy-nested-owners.json")
	if e != nil {
		f.Fatal(e)
	}
	f.Add(b)
	f.Add(schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag"}},"Leaf":{"properties":{"node":{"properties":{"child":{"$ref":"#/components/schemas/Leaf/properties/node"}}}}},"Alias":{"$ref":"#/components/schemas/Leaf/properties/node"}}`))
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
				t.Fatal("nondeterministic ownership")
			}
			if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial failed report")
			}
		}
	})
}
