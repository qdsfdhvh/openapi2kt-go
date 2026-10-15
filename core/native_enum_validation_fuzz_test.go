package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNativeEnumDTOChecks(f *testing.F) {
	f.Add(int64(1), int64(2), false)
	f.Add(int64(-2147483648), int64(2147483647), true)
	f.Fuzz(func(t *testing.T, first, second int64, nullable bool) {
		first = int64(int32(first))
		second = int64(int32(second))
		if first == second {
			return
		}
		doc := []byte(fmt.Sprintf(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"value":{"type":"integer","nullable":%t,"enum":[%d,%d]}}}}}}`, nullable, first, second))
		r := request(doc, "Dto")
		r.Options.Validation = true
		a, err := core.GenerateWithDiagnostics(r)
		b, e := core.GenerateWithDiagnostics(r)
		if err != nil || e != nil || !reflect.DeepEqual(a, b) {
			t.Fatal(err, e)
		}
	})
}
