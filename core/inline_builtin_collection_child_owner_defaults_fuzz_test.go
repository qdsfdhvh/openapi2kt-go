package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzInlineBuiltinCollectionChildOwnerDefault(f *testing.F) {
	for _, value := range []string{"", "value", "\"\\/\n", "\x00", "一"} {
		f.Add(value, false, false, false, false)
		f.Add(value, true, true, true, true)
	}
	f.Fuzz(func(t *testing.T, value string, dictionary, invalid, nullableOwner, checked bool) {
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
		ownerType, ownerValue := `"type":"object",`, `{}`
		if nullableOwner {
			ownerType, ownerValue = `"type":["object","null"],`, `null`
		}
		before := `{"Holder":{` + ownerType + `"properties":{"value":{` + prefix + child + `:{"type":"string"}}},"default":` + ownerValue + `}}`
		after := `{"Holder":{` + ownerType + `"properties":{"value":{` + prefix + child + `:{"type":"string","default":` + annotation + `}}},"default":` + ownerValue + `}}`
		rq := request(schemas(before), "Dto")
		rq.Options.Validation = checked
		control, failure := core.GenerateWithDiagnostics(rq)
		if failure != nil {
			t.Fatal("unannotated owner default rejected", failure)
		}
		rq.Document = schemas(after)
		got, failure := core.GenerateWithDiagnostics(rq)
		if !invalid {
			if failure != nil || !reflect.DeepEqual(got, control) {
				t.Fatal("child annotation changed owner default", failure)
			}
			return
		}
		ds, ok := failure.(core.Diagnostics)
		want := "/components/schemas/Holder/properties/value" + suffix
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
			*ds[0].Pointer != want || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("invalid child annotation escaped owner-default validation", failure)
		}
	})
}
