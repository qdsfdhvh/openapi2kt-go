package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

// Exercise retained and discarded constructor sources, deferred reference
// annotations and suppression with bounded original trees, not arbitrary bytes.
func FuzzTupleInlineDTO(f *testing.F) {
	for _, mode := range []uint8{0, 1, 2, 3, 4, 7, 12, 16, 31} {
		f.Add(mode, false, true)
	}
	f.Fuzz(func(t *testing.T, mode uint8, nullable, checked bool) {
		field := map[string]any{"$ref": "#/components/schemas/Choice", "default": "ready"}
		if mode&4 != 0 {
			field["default"] = "future"
		}
		member := map[string]any{"properties": map[string]any{"state": field, "name": map[string]any{"type": "string"}}, "required": []any{"name"}}
		if nullable {
			member["nullable"] = true
		}
		tuple := map[string]any{"type": "array", "prefixItems": []any{member}}
		switch mode & 3 {
		case 1:
			tuple["items"] = true
		case 2:
			tuple["items"] = false
		case 3:
			tuple["items"] = map[string]any{"type": "string"}
		}
		if mode&8 != 0 {
			tuple["default"] = []any{}
		}
		if mode&16 != 0 {
			tuple["nullable"] = true
			tuple["default"] = nil
		}
		literal, e := json.Marshal(tuple)
		if e != nil {
			t.Fatal(e)
		}
		for _, required := range []bool{false, true} {
			owner := map[string]any{"properties": map[string]any{"rows": tuple}}
			if required {
				owner["required"] = []any{"rows"}
			}
			holder, _ := json.Marshal(owner)
			input := schemas(`{"Choice":{"enum":["ready"]},"Rows":` + string(literal) + `,"Holder":` + string(holder) + `}`)
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			again, repeated := core.GenerateWithDiagnostics(rq)
			if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(err, repeated) {
				t.Fatal("inline DTO tuple nondeterminism", mode)
			}
			if err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("partial output after original DTO source failure")
			}
			if (err != nil) != (mode&4 != 0) {
				t.Fatal("discarded/suppressed deferred annotation", mode, required, err)
			}
		}
	})
}
