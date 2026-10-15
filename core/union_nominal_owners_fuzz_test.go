package core_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzUnionNominalOwners(f *testing.F) {
	b, e := os.ReadFile("../tests/fixtures/http/union-nominal-owners.json")
	if e != nil {
		f.Fatal(e)
	}
	f.Add(b)
	f.Add(schemas(`{"Owner":{"properties":{"choice":{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"child":{"$ref":"#/components/schemas/Owner/properties/choice"}}},{"properties":{"kind":{"type":"string","const":"b"},"name":{"type":"string"}}},{"type":"null"}]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/choice","nullable":false}}`))
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
				t.Fatal("nondeterministic nominal union")
			}
			if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial failed report")
			}
		}
	})
}
