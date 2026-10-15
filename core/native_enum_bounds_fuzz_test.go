package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNativeEnumBoundIntersection(f *testing.F) {
	f.Add(int32(1), int32(2), int32(2))
	f.Add(int32(-2147483648), int32(2147483647), int32(0))
	f.Fuzz(func(t *testing.T, first, second, minimum int32) {
		if first == second {
			return
		}
		doc := fieldDoc(fmt.Sprintf(`{"type":"integer","enum":[%d,%d],"minimum":%d}`, first, second, minimum))
		r := request(doc, "Dto")
		r.Options.Validation = true
		a, err := core.GenerateWithDiagnostics(r)
		b, e := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(err, e) {
			t.Fatal("nondeterministic", err, e)
		}
		possible := first >= minimum || second >= minimum
		if (err == nil) != possible {
			t.Fatal(first, second, minimum, err)
		}
		if err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || !reflect.DeepEqual(a, core.GenerationReport{})) {
			t.Fatal(err)
		}
	})
}
