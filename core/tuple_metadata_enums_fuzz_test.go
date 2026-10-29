package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleMetadataEnums(f *testing.F) {
	f.Add("ready", uint8(0), false)
	f.Add("null", uint8(7), true)
	f.Add("future", uint8(18), true)
	f.Fuzz(func(t *testing.T, wire string, mode uint8, checked bool) {
		if len(wire) > 96 {
			t.Skip()
		}
		var metadata map[string]any
		if err := json.Unmarshal([]byte(`{`+tupleStringMetadata[int(mode)%len(tupleStringMetadata)]+`}`), &metadata); err != nil {
			t.Fatal(err)
		}
		metadata["type"], metadata["enum"] = "string", []any{wire}
		var selected any = metadata
		if mode&1 != 0 {
			selected = map[string]any{"$ref": "#/components/schemas/Value"}
		}
		array := map[string]any{"type": "array", "prefixItems": []any{selected}, "default": []any{wire}}
		if mode&2 != 0 {
			array["items"] = true
			array["default"] = []any{nil, map[string]any{"enum": wire}}
		}
		invalid := mode&4 != 0
		if invalid {
			array["default"] = false
		}
		data, _ := json.Marshal(map[string]any{"Value": metadata, "Rows": array, "Holder": map[string]any{"properties": map[string]any{"rows": array}, "required": []any{"rows"}}})
		rq := request(schemas(string(data)), "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(err, repeated) {
			t.Fatal("metadata tuple nondeterminism")
		}
		if (err != nil) != invalid {
			t.Fatal("metadata vocabulary/default contract", mode, err)
		}
		if err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("partial metadata tuple output")
		}
	})
}
