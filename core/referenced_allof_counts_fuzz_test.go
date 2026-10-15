package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzReferencedNestedAllOfCounts(f *testing.F) {
	f.Add(uint16(0), uint16(2), false)
	f.Add(uint16(65535), uint16(65535), true)
	f.Fuzz(func(t *testing.T, minimum, width uint16, nested bool) {
		member := `{"$ref":"#/components/schemas/Lower"}`
		if nested {
			member = `{"type":"object","allOf":[` + member + `]}`
		}
		r := request(schemas(fmt.Sprintf(`{"Lower":{"properties":{"value":{"type":"integer"}},"minProperties":%d},"Record":{"allOf":[%s,{"properties":{"value":{"type":"integer"}},"maxProperties":%d}]}}`, minimum, member, int(minimum)+int(width))), "Dto")
		r.Options.Validation = true
		a, e := core.GenerateWithDiagnostics(r)
		b, err := core.GenerateWithDiagnostics(r)
		if e != nil || err != nil || !reflect.DeepEqual(a, b) {
			t.Fatal(e, err)
		}
	})
}
