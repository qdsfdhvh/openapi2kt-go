package core_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzRootBorrowedBuiltin(f *testing.F) {
	b, err := os.ReadFile("../tests/fixtures/http/root-borrowed-builtin.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(b)
	f.Add(schemas(`{"Source":{"properties":{"leaf":{"type":["integer","null"],"format":"int64"}}},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/leaf"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 100000 {
			t.Skip()
		}
		for _, checked := range []bool{false, true} {
			r := request(b, "Dto")
			r.Options.Validation = checked
			a, err := core.GenerateWithDiagnostics(r)
			c, e := core.GenerateWithDiagnostics(r)
			if !reflect.DeepEqual(a, c) || fmt.Sprint(err) != fmt.Sprint(e) {
				t.Fatal("nondeterministic borrowed builtin source graph")
			}
			if err != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial report on failed builtin source")
			}
		}
	})
}
