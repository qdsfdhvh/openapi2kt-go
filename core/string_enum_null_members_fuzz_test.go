package core_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzStringEnumNullMembers(f *testing.F) {
	b, e := os.ReadFile("../tests/fixtures/http/string-enum-null-members.json")
	if e != nil {
		f.Fatal(e)
	}
	f.Add(b)
	f.Add(schemas(`{"Owner":{"properties":{"state":{"enum":["on",null,"off"],"nullable":true}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state","nullable":false}}`))
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
				t.Fatal("nondeterministic nominal vocabulary")
			}
			if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial failed artifacts")
			}
		}
	})
}
