package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStructuredConstFieldsRetainSerializedSlotsAndSchemaDepth(t *testing.T) {
	for _, value := range []string{`{"const":{"n":1}}`, `{"type":"array","const":[null]}`} {
		for _, width := range []int{240, 241} {
			fields := []string{}
			for i := range width {
				fields = append(fields, fmt.Sprintf(`"p%d":%s`, i, value))
			}
			got, err := core.GenerateWithDiagnostics(request(schemas(`{"Wide":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
			if (err == nil) != (width == 240) || err != nil && (err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
				t.Fatal("Structured body serialization slots changed", value, width, err)
			}
		}
	}
	for _, depth := range []int{47, 48} {
		value := `{"properties":{"value":{"const":[{"$ref":"data"}]}}}`
		for i := range depth {
			value = fmt.Sprintf(`{"properties":{"n%d":%s}}`, i, value)
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":`+value+`}`), "Dto"))
		if (err == nil) != (depth == 47) || err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Structured body schema source depth changed", depth, err)
		}
	}
}

func TestStructuredConstFieldsRetainOwnedDefaultsAndShapeLockIdentity(t *testing.T) {
	op := func(id, literal string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"kotlinx":{"type":"string"},"payload":{"const":` + literal + `}}}}}}}}}`
	}
	input := []byte(`{"openapi":"3.1.0","paths":{"/a":` + op("alpha", `{"n":1}`) + `,"/b":` + op("beta", `{"n":1}`) + `,"/c":` + op("gamma", `{"n":2}`) + `}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, qualified := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.ModelStyle = style
			r.Options.HoistSharedShapes, r.Options.DeduplicateModels, r.Options.MergeIdenticalModels, r.Options.LockHoistedNames = true, true, true, true
			r.Options.QualifyHoistedNames = qualified
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "AlphaResponseDto"), "typealias") || strings.Contains(source(t, got, "GammaResponseDto"), "typealias") {
				t.Fatal("Different structured defaults merged", style, qualified, err)
			}
			r.PreviousNameLock = got.Artifacts.NameLock
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Structured body owned lock replay drifted", err)
			}
			r.Document = []byte(strings.ReplaceAll(string(input), `"n":1`, `"n":3`))
			changed, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
				t.Fatal("Changed structured default lost lock identity", err)
			}
		}
	}
}
