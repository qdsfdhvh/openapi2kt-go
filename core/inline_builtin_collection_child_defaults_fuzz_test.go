package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzInlineBuiltinCollectionChildDefault(f *testing.F) {
	for _, value := range []string{"", "ordinary", "\"\\/\n", "\x00", "一", "null"} {
		for _, dictionary := range []bool{false, true} {
			f.Add(value, false, dictionary, false)
			f.Add(value, true, dictionary, true)
		}
	}
	f.Fuzz(func(t *testing.T, value string, checked, dictionary, invalid bool) {
		if len(value) > 256 {
			t.Skip()
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		annotation := string(encoded)
		if invalid {
			annotation = "true"
		}
		child, suffix := `"items"`, "/items/default"
		prefix := `"type":"array",`
		if dictionary {
			child, suffix = `"additionalProperties"`, "/additionalProperties/default"
			prefix = `"type":"object",`
		}
		before := `{"Holder":{"properties":{"value":{` + prefix + child + `:{"type":"string"}}}}}`
		after := `{"Holder":{"properties":{"value":{` + prefix + child + `:{"type":"string","default":` + annotation + `}}}}}`
		rq := request(schemas(before), "Dto")
		rq.Options.Validation = checked
		control, failure := core.GenerateWithDiagnostics(rq)
		if failure != nil {
			t.Fatal("unannotated control rejected", failure)
		}
		rq.Document = schemas(after)
		got, failure := core.GenerateWithDiagnostics(rq)
		if !invalid {
			if failure != nil || !reflect.DeepEqual(got, control) {
				t.Fatal("valid child annotation changed generation", failure)
			}
			return
		}
		ds, ok := failure.(core.Diagnostics)
		want := "/components/schemas/Holder/properties/value" + suffix
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
			*ds[0].Pointer != want || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("invalid child annotation escaped validation", failure)
		}
	})
}
