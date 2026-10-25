package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzScalarTupleRootDefaults(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(7), true, true)
	f.Fuzz(func(t *testing.T, count uint8, bad, nullable bool) {
		prefix := []any{}
		for i := 0; i < int(count%8)+1; i++ {
			prefix = append(prefix, map[string]any{"type": "integer"})
		}
		var literal any = []any{1, 2}
		if nullable {
			literal = nil
		}
		if bad {
			literal = []any{true}
		}
		input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Rows": map[string]any{"type": "array", "nullable": nullable, "prefixItems": prefix, "default": literal}}}})
		if e != nil {
			t.Fatal(e)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = count%2 == 0
		first, e := core.GenerateWithDiagnostics(rq)
		if bad {
			if e == nil || len(first.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Rows/default/0" {
				t.Fatal("bad tuple literal accepted", e)
			}
			return
		}
		if e != nil {
			t.Fatal("valid tuple root rejected", e)
		}
		second, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("tuple root nondeterminism", e)
		}
	})
}
