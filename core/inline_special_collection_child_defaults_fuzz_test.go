package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

// Compare the new annotation role with the existing collection-default
// grammar for the same string and mapped Kotlin scalar.
func FuzzInlineSpecialCollectionChildDefault(f *testing.F) {
	for _, seed := range []struct {
		format uint8
		value  string
	}{
		{0, "123E4567E89B12D3A456426614174000"},
		{0, "not-a-uuid"},
		{1, "2024-02-29"},
		{1, "2023-02-29"},
		{2, "2024-02-29T12:34:56.123456789+18:00"},
		{2, "2024-02-29T12:34:56.1234567890Z"},
	} {
		f.Add(seed.format, seed.value)
	}
	f.Fuzz(func(t *testing.T, formatIndex uint8, value string) {
		if len(value) > 256 {
			t.Skip()
		}
		formats := []string{"uuid", "date", "date-time"}
		format := formats[int(formatIndex)%len(formats)]
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		child := `{"type":"string","format":"` + format + `"}`
		before := `{"Holder":{"properties":{"value":{"type":"array","items":` + child + `}}}}`
		annotated := `{"Holder":{"properties":{"value":{"type":"array","items":{"type":"string","format":"` +
			format + `","default":` + string(encoded) + `}}}}}`
		outer := `{"Holder":{"properties":{"value":{"type":"array","items":` + child + `,"default":[` + string(encoded) + `]}}}}`
		rq := request(schemas(before), "Dto")
		control, failure := core.GenerateWithDiagnostics(rq)
		if failure != nil {
			t.Fatal("unannotated control rejected", failure)
		}
		rq.Document = schemas(outer)
		_, outerFailure := core.GenerateWithDiagnostics(rq)
		rq.Document = schemas(annotated)
		got, childFailure := core.GenerateWithDiagnostics(rq)
		if outerFailure == nil {
			if childFailure != nil || !reflect.DeepEqual(got, control) {
				t.Fatal("child annotation disagrees with accepted collection member", format, value, childFailure)
			}
			return
		}
		outerDiagnostics, outerOK := outerFailure.(core.Diagnostics)
		childDiagnostics, childOK := childFailure.(core.Diagnostics)
		want := "/components/schemas/Holder/properties/value/items/default"
		if !outerOK || len(outerDiagnostics) == 0 || outerDiagnostics[0].Code != "E_SCHEMA_INVALID" ||
			!childOK || len(childDiagnostics) == 0 || childDiagnostics[0].Code != "E_SCHEMA_INVALID" ||
			childDiagnostics[0].Pointer == nil || *childDiagnostics[0].Pointer != want ||
			!reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("child annotation disagrees with rejected collection member", format, value, outerFailure, childFailure)
		}
	})
}
