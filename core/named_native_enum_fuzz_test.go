package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNamedNativeEnumBorrowing(f *testing.F) {
	f.Add(int32(1), int32(2), false)
	f.Add(int32(-2147483648), int32(2147483647), true)
	f.Fuzz(func(t *testing.T, first, second int32, nullable bool) {
		if first == second {
			return
		}
		input := schemas(fmt.Sprintf(`{"Code":{"type":"integer","enum":[%d,%d]},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop","nullable":%t},"Rows":{"$ref":"#/components/schemas/Source/properties/rows"},"Source":{"properties":{"hop":{"$ref":"#/components/schemas/Code"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Code"}}}},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Bridge"},"rows":{"$ref":"#/components/schemas/Rows"}}}}`, first, second, nullable))
		r := request(input, "Dto")
		r.Options.Validation = true
		a, e := core.GenerateWithDiagnostics(r)
		b, err := core.GenerateWithDiagnostics(r)
		if e != nil || err != nil || !reflect.DeepEqual(a, b) {
			t.Fatal(e, err)
		}
	})
}
