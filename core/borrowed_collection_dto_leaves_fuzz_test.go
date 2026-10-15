package core_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzBorrowedCollectionDTOLeaves(f *testing.F) {
	b, err := os.ReadFile("../tests/fixtures/http/borrowed-collection-dto-leaves.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(b)
	f.Add(schemas(`{"Source":{"properties":{"value":{"type":"array","nullable":true,"items":{"type":"string"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/value","nullable":false}}`))
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
				t.Fatal("nondeterministic borrowed root collection graph")
			}
			if err != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial report on failed borrowed root collection graph")
			}
		}
	})
}
