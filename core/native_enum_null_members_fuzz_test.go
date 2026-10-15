package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNativeEnumNullMemberChecks(f *testing.F) {
	f.Add(int32(1), int32(2), false)
	f.Add(int32(-2147483648), int32(2147483647), true)
	f.Fuzz(func(t *testing.T, first, second int32, narrow bool) {
		if first == second {
			return
		}
		input := schemas(fmt.Sprintf(`{"Code":{"type":["integer","null"],"enum":[%d,null,%d]},"Alias":{"$ref":"#/components/schemas/Code","nullable":%t},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias"}}},"Source":{"properties":{"hop":{"$ref":"#/components/schemas/Alias"}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"}}`, first, second, !narrow))
		r := request(input, "Dto")
		r.Options.Validation = true
		a, e := core.GenerateWithDiagnostics(r)
		b, err := core.GenerateWithDiagnostics(r)
		if e != nil || err != nil || !reflect.DeepEqual(a, b) {
			t.Fatal(e, err)
		}
	})
}
