package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullOnlyDefaultsValidateOriginalDiscardedFields(t *testing.T) {
	for _, tc := range []struct{ input, pointer string }{
		{`{"Holder":{"allOf":[{"properties":{"value":{"type":"null","default":false}}},{"properties":{"value":{"type":"string"}}}]}}`, "/components/schemas/Holder/allOf/0/properties/value/default"},
		{`{"Holder":{"allOf":[{"properties":{"value":{"enum":[null,null],"default":null}}},{"properties":{"value":{"type":"string"}}}]}}`, "/components/schemas/Holder/allOf/0/properties/value/enum/1"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(schemas(tc.input), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != tc.pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("A discarded null default source was erased", tc, validation, err)
			}
		}
	}
}

func TestNullOnlyDefaultsRetainModelAndSourceBudgets(t *testing.T) {
	for _, field := range []string{`{"type":"null","default":null}`, `{"enum":[null],"default":null}`} {
		for _, width := range []int{240, 241} {
			properties, required := []string{}, []string{}
			for i := range width {
				properties = append(properties, fmt.Sprintf(`"p%d":%s`, i, field))
				required = append(required, fmt.Sprintf(`"p%d"`, i))
			}
			r := request(schemas(`{"Holder":{"properties":{`+strings.Join(properties, ",")+`},"required":[`+strings.Join(required, ",")+`]}}`), "Dto")
			got, err := core.GenerateWithDiagnostics(r)
			if (err == nil) != (width == 240) || err != nil && (err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
				t.Fatal("Null default altered constructor slots", field, width, err)
			}
		}
	}
	for _, depth := range []int{47, 48} {
		value := `{"properties":{"value":{"enum":[null],"default":null}}}`
		for i := range depth {
			value = fmt.Sprintf(`{"properties":{"n%d":%s}}`, i, value)
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":`+value+`}`), "Dto"))
		if (err == nil) != (depth == 47) || err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Null default erased original schema depth", depth, err)
		}
	}
}

func TestNullOnlyDefaultsRetainOwnedReuseAndLockReplay(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, qualified := range []bool{false, true} {
			input := []byte(strings.ReplaceAll(string(explicitNullOperations(`{"const":null}`)), `{"type":"null","enum":[null]}`, `{"enum":[null],"default":null}`))
			r := request(input, "Dto")
			r.Options.Validation = true
			r.Options.ModelStyle = style
			r.Options.HoistSharedShapes, r.Options.DeduplicateModels, r.Options.MergeIdenticalModels, r.Options.LockHoistedNames = true, true, true, true
			r.Options.QualifyHoistedNames = qualified
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, qualified, err)
			}
			r.PreviousNameLock = got.Artifacts.NameLock
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Null default owned lock replay drifted", err)
			}
			r.Document = []byte(strings.ReplaceAll(string(input), `{"enum":[null],"default":null}`, `{"type":"null","default":null}`))
			control, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, control) {
				t.Fatal("Equivalent null default carrier changed lock identity", err)
			}
		}
	}
}
