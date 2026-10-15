package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzExplicitNullLiterals(f *testing.F) {
	f.Add(fieldDoc(`{"type":"null","const":null}`))
	f.Add(fieldDoc(`{"type":"null","enum":[null]}`))
	f.Add(fieldDoc(`{"type":"null","enum":[null,null]}`))
	f.Add(fieldDoc(`{"const":null}`))
	f.Add(fieldDoc(`{"enum":[null]}`))
	f.Add(fieldDoc(`{"enum":[null,null]}`))
	f.Add(fieldDoc(`{"type":"null","default":null}`))
	f.Add(fieldDoc(`{"enum":[null],"default":null}`))
	f.Add(fieldDoc(`{"enum":[null],"default":false}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 100000 {
			t.Skip()
		}
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			a, e := core.GenerateWithDiagnostics(r)
			b, err := core.GenerateWithDiagnostics(r)
			if !reflect.DeepEqual(a, b) || fmt.Sprint(e) != fmt.Sprint(err) {
				t.Fatal("Nondeterministic null literal profile")
			}
			if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("Partial failed null artifacts")
			}
		}
	})
}
