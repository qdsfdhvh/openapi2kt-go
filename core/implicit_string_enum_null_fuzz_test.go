package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzImplicitStringEnumNull(f *testing.F) {
	f.Add("on", "off", uint8(0))
	f.Add("null", "future", uint8(15))
	f.Add("same", "same", uint8(4))
	f.Add("$ref", "type", uint8(7))
	f.Fuzz(func(t *testing.T, first, second string, mode uint8) {
		if len(first)+len(second) > 512 {
			t.Skip()
		}
		members, _ := json.Marshal([]any{first, nil, second})
		body := `{"State":{"enum":` + string(members) + `},"Alias":{"$ref":"#/components/schemas/State"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/State"}}},"required":["value"]}}`
		r := request(schemas(body), "Dto")
		r.Options.Validation = mode&1 != 0
		if mode&2 != 0 {
			r.Options.ModelStyle = core.PlainClass
		}
		r.Options.DeduplicateModels, r.Options.HoistSharedShapes, r.Options.MergeIdenticalModels = mode&4 != 0, mode&4 != 0, mode&4 != 0
		got, err := core.GenerateWithDiagnostics(r)
		// JSON string encoding normalizes invalid UTF-8 before vocabulary checks.
		var encoded []any
		if e := json.Unmarshal(members, &encoded); e != nil {
			t.Fatal(e)
		}
		if encoded[0] == encoded[2] {
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("duplicate original String vocabulary admitted", err)
			}
			return
		}
		if err != nil {
			t.Fatal("valid implicit vocabulary refused", err)
		}
		r.Document = schemas(strings.ReplaceAll(body, `"enum":`, `"nullable":true,"enum":`))
		control, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("explicit nullable control drift", err)
		}
		r.Document = schemas(strings.ReplaceAll(body, `"enum":`, `"nullable":false,"enum":`))
		failed, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(failed, core.GenerationReport{}) {
			t.Fatal("explicit false lost failure atomicity", err)
		}
	})
}
