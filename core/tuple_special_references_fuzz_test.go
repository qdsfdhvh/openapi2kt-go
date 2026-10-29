package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleSpecialReferences(f *testing.F) {
	for index := range tupleSpecialCases {
		kind := uint8(index)
		for _, mode := range []uint8{0, 1, 2, 4, 8, 16, 32, 64, 128, 255} {
			f.Add(kind, mode, false, true)
		}
	}
	f.Fuzz(func(t *testing.T, kind, mode uint8, nullable, checked bool) {
		tc := tupleSpecialCases[int(kind)%len(tupleSpecialCases)]
		var source map[string]any
		var value any
		if err := json.Unmarshal([]byte(tc.source), &source); err != nil {
			t.Fatal(err)
		}
		decoder := json.NewDecoder(strings.NewReader(tc.value))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		if (!nullable || mode&128 != 0) && tc.name != "raw" {
			members := []any{}
			for _, member := range value.([]any) {
				if member != nil {
					members = append(members, member)
				}
			}
			value = members
		}
		catalog := map[string]any{"Value": source, "Maybe": map[string]any{"$ref": "#/components/schemas/Value", "nullable": true}}
		target := "Value"
		if mode&1 != 0 {
			target = "Maybe"
		}
		member := map[string]any{"$ref": "#/components/schemas/" + target, "nullable": nullable}
		array := map[string]any{"type": "array", "prefixItems": []any{member}, "default": value}
		if mode&2 != 0 {
			array["items"] = true
		}
		if mode&4 != 0 {
			array["items"], array["default"] = map[string]any{"type": "integer"}, []any{7}
		}
		if mode&8 != 0 {
			array["prefixItems"] = []any{member, map[string]any{"type": "boolean"}}
		}
		invalid := mode&112 != 0
		if mode&16 != 0 {
			member["default"] = nil
		}
		if mode&32 != 0 {
			source["unknownTupleRole"] = true
		}
		if mode&64 != 0 {
			array["default"] = 7
		}
		if mode&128 != 0 {
			member["nullable"] = false
		}
		catalog["Rows"] = array
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, failure := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(failure, repeated) || (failure != nil) != invalid || failure != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc.name, mode, nullable, checked, failure, repeated)
		}
	})
}
