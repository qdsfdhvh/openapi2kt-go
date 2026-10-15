package core_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzComposedNestedOwners(f *testing.F) {
	b, e := os.ReadFile("../tests/fixtures/http/composed-nested-owners.json")
	if e != nil {
		f.Fatal(e)
	}
	f.Add(b)
	f.Add(schemas(`{"Owner":{"allOf":[{"properties":{"node":{"properties":{"self":{"$ref":"#/components/schemas/Owner/allOf/0/properties/node"}}}}},{"properties":{"extra":{"type":"string"}}}]},"Alias":{"$ref":"#/components/schemas/Owner/allOf/0/properties/node"}}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 100000 {
			t.Skip()
		}
		for _, checked := range []bool{false, true} {
			req := request(b, "Dto")
			req.Options.Validation = checked
			a, e := core.GenerateWithDiagnostics(req)
			c, e2 := core.GenerateWithDiagnostics(req)
			if !reflect.DeepEqual(a, c) || fmt.Sprint(e) != fmt.Sprint(e2) {
				t.Fatal("nondeterministic source ownership")
			}
			if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial failed generation")
			}
		}
	})
}
