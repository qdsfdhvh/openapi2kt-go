package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzCollectionReferenceChildDefault(f *testing.F) {
	for _, value := range []string{"one", "two", "", "future", "\"", "null", "\u0000", "一"} {
		f.Add(value, false)
		f.Add(value, true)
	}
	f.Fuzz(func(t *testing.T, value string, checked bool) {
		if len(value) > 256 {
			t.Skip()
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		doc := `{"Word":{"enum":["one","two"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Word","default":` + string(encoded) + `},"default":[]}}`
		rq := request(schemas(doc), "Dto")
		rq.Options.Validation = checked
		report, failure := core.GenerateWithDiagnostics(rq)
		if value == "one" || value == "two" {
			if failure != nil {
				t.Fatal("valid vocabulary member rejected", failure)
			}
			return
		}
		ds, ok := failure.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
			*ds[0].Pointer != "/components/schemas/Rows/items/default" ||
			!reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("invalid child default escaped validation", failure)
		}
	})
}
