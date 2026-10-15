package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzInternalNativeEnumPointerChecks(f *testing.F) {
	f.Add(int32(1), int32(2), false)
	f.Add(int32(-2147483648), int32(2147483647), true)
	f.Fuzz(func(t *testing.T, first, second int32, nullable bool) {
		if first == second {
			return
		}
		input := schemas(fmt.Sprintf(`{"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf","nullable":%t},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Bridge"}}},"Source":{"properties":{"leaf":{"type":"integer","enum":[%d,%d]}}}}`, nullable, first, second))
		r := request(input, "Dto")
		r.Options.Validation = true
		a, e := core.GenerateWithDiagnostics(r)
		b, err := core.GenerateWithDiagnostics(r)
		if e != nil || err != nil || !reflect.DeepEqual(a, b) {
			t.Fatal(e, err)
		}
	})
}
