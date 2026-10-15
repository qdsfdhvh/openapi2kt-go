package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzReferencedNestedAllOfBounds(f *testing.F) {
	f.Add(int32(1), uint16(2), false)
	f.Add(int32(-2147483648), uint16(65535), true)
	f.Fuzz(func(t *testing.T, lower int32, width uint16, nested bool) {
		upper := int64(lower) + int64(width)
		if upper > 2147483647 {
			return
		}
		member := `{"$ref":"#/components/schemas/Lower"}`
		if nested {
			member = `{"type":"object","allOf":[` + member + `]}`
		}
		input := schemas(fmt.Sprintf(`{"Lower":{"properties":{"value":{"type":"integer","minimum":%d}}},"Record":{"allOf":[%s,{"properties":{"value":{"type":"integer","maximum":%d}}}]}}`, lower, member, upper))
		r := request(input, "Dto")
		r.Options.Validation = true
		a, e := core.GenerateWithDiagnostics(r)
		b, err := core.GenerateWithDiagnostics(r)
		if e != nil || err != nil || !reflect.DeepEqual(a, b) {
			t.Fatal(e, err)
		}
	})
}
