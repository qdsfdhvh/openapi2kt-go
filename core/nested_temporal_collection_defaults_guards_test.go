package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedTemporalCollectionDefaultsKeepExcludedRolesAndNoDrop(t *testing.T) {
	for _, tc := range []struct{ leaf, tail string }{
		{`{"type":"string","format":"date","minLength":1}`, "/minLength"},
		{`{"type":"string","format":"date-time","minimum":0}`, "/minimum"},
		{`{"type":"string","format":"date","pattern":".*"}`, "/pattern"},
		{`{"type":"string","format":"date","enum":["2024-02-29"]}`, "/enum"},
		{`{"type":"string","format":"date","const":"2024-02-29"}`, "/const"},
		{`{"type":"string","x-native-type":"date"}`, "/x-native-type"},
		{`{"type":"string","x-native-type":"org.example.Time"}`, "/x-native-type"},
		{`{"type":"string","format":"date","x-kotlin-value-class":false}`, "/x-kotlin-value-class"},
		{`{"$ref":"#/components/schemas/Day"}`, "/$ref"},
		{`{"type":"string","format":"date","default":"2024-02-29"}`, "/default"},
		{`{"oneOf":[{"type":"string","format":"date"},{"type":"null"}]}`, "/oneOf"},
	} {
		field := `{"type":"array","items":{"additionalProperties":` + tc.leaf + `},"default":[]}`
		for _, checked := range []bool{false, true} {
			r := request(fieldDoc(field), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value/items/additionalProperties"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, checked, err)
			}
		}
	}
	field := `{"type":"array","items":{"type":"array","items":{"type":"string","format":"date"}},"default":[]}`
	for _, input := range []string{
		`{"Root":` + field + `}`,
		`{"Root":{"type":"array","items":` + field + `}}`,
		`{"Root":{"additionalProperties":` + field + `}}`,
		`{"Day":{"type":"string","format":"date"},"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"$ref":"#/components/schemas/Day"}},"default":[]}}}}`,
		`{"Holder":{"properties":{"value":{"allOf":[` + field + `]}}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":` + field + `}},{"properties":{"tag":{"const":"two"}}}]}}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"properties":{"day":{"type":"string","format":"date"}}}},"default":[]}}}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(input), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("excluded role leaked", input, checked, err)
			}
		}
	}
	for _, field := range []string{
		`{"type":"array","items":{"type":"array","items":{"type":"string","format":"date"},"uniqueItems":true},"default":[]}`,
		`{"type":"array","items":{"type":"array","items":{"type":"string","format":"date"},"default":[]},"default":[]}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(fieldDoc(field), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err == nil {
				t.Fatal("unsupported child/temporal uniqueness leaked", field)
			}
		}
	}
	bad := strings.Replace(field, `"default":[]`, `"default":[["2023-02-29"]]`, 1)
	for _, input := range [][]byte{
		schemas(`{"Holder":{"properties":{"value":` + bad + `},"required":["value"]}}`),
		schemas(`{"Holder":{"allOf":[{"properties":{"value":` + bad + `}},{"properties":{"value":{"type":"string"}}}]}}`),
		[]byte(`{"openapi":"3.1.0","paths":{"/patch":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + strings.Replace(bad, `"type":"array"`, `"type":"array","nullable":true`, 1) + `}}}}},"responses":{"204":{"description":"OK"}}}}}}`),
	} {
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || !strings.HasSuffix(*err.(core.Diagnostics)[0].Pointer, "/default/0/0") || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("required/Patch/discarded validation suppressed", checked, err)
			}
		}
	}
}

func TestNestedTemporalCollectionDefaultsDeepCountsAndDiscardedChecks(t *testing.T) {
	for _, tc := range []struct{ field, tail string }{
		{`{"type":"array","items":{"type":"array","minItems":1,"items":{"type":"string","format":"date"}},"default":[[]]}`, "/default/0"},
		{`{"additionalProperties":{"additionalProperties":{"type":"string","format":"date-time"},"maxProperties":0},"default":{"a/~":{"b":"2024-02-29T00:00:00Z"}}}`, "/default/a~1~0"},
		{`{"type":"array","items":{"additionalProperties":{"type":"string","format":"date"},"minProperties":1},"default":[{}]}`, "/default/0"},
	} {
		r := request(fieldDoc(tc.field), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc, err)
		}
		r.Options.Validation = false
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("flag-disabled counts leaked", tc)
		}
	}
	field := `{"type":"array","items":{"type":"array","minItems":1,"items":{"type":"string","format":"date"}},"default":[["2024-02-29"]]}`
	r := request(schemas(`{"Holder":{"allOf":[{"properties":{"value":`+field+`}},{"properties":{"value":{"type":"string"}}}]}}`), "Dto")
	r.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(r); err == nil {
		t.Fatal("discarded deep check lost")
	}
}

func TestNestedTemporalCollectionDefaultsCanonicalLockAndActualOptimization(t *testing.T) {
	field := `{"additionalProperties":{"additionalProperties":{"type":"string","nullable":true,"format":"date-time"}},"default":{"z":{"z":null},"a":{"z":"2024-02-29T12:34:56+02:30","a":"2024-02-29T00:00:00Z"}}}`
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			base := field
			if checked {
				base = strings.Replace(base, `"additionalProperties":{"additionalProperties"`, `"additionalProperties":{"maxProperties":3,"additionalProperties"`, 1)
			}
			r := request(nestedBuiltinDefaultOperations(base), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			r.Options.HoistSharedShapes, r.Options.LockHoistedNames, r.Options.DeduplicateModels, r.Options.MergeIdenticalModels = true, true, true, true
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || len(got.Artifacts.NameLock) == 0 || !strings.Contains(source(t, got, "ResponseDto"), "serializer<kotlin.collections.Map<kotlin.String, kotlin.collections.Map<kotlin.String, kotlin.time.Instant?>>>") || !strings.Contains(source(t, got, "AlphaResponseDto"), "typealias AlphaResponseDto = org.example.models.models.ResponseDto") {
				t.Fatal("actual temporal type lost in opt/lock", err)
			}
			r.PreviousNameLock = got.Artifacts.NameLock
			r.Document = []byte(strings.ReplaceAll(string(r.Document), `{"z":{"z":null},"a":{"z":"2024-02-29T12:34:56+02:30","a":"2024-02-29T00:00:00Z"}}`, `{"a":{"a":"2024-02-29T00:00:00Z","z":"2024-02-29T12:34:56+02:30"},"z":{"z":null}}`))
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("canonical-key lock replay changed", err)
			}
			for _, change := range []string{
				strings.Replace(base, "2024-02-29T12:34:56+02:30", "2024-02-29T10:04:56Z", 1),
				strings.NewReplacer("date-time", "date", "2024-02-29T12:34:56+02:30", "2024-02-29", "2024-02-29T00:00:00Z", "2024-02-29").Replace(base),
				strings.Replace(base, `"additionalProperties":{"additionalProperties"`, `"additionalProperties":{"nullable":true,"additionalProperties"`, 1),
				strings.Replace(base, `"maxProperties":3`, `"maxProperties":4`, 1),
			} {
				if change == base {
					continue
				}
				r.Document = nestedBuiltinDefaultOperations(change)
				changed, err := core.GenerateWithDiagnostics(r)
				if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
					t.Fatal("temporal lexeme/check changed frozen identity", err)
				}
			}
		}
	}
	for _, mode := range []string{"dedup", "merge", "combined"} {
		r := request(nestedBuiltinDefaultOperations(field), "Dto")
		r.Options.DeduplicateModels, r.Options.MergeIdenticalModels, r.Options.HoistSharedShapes = mode != "merge", mode != "dedup", mode == "combined"
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "BetaResponseDto"), "typealias BetaResponseDto") {
			t.Fatal("equivalent temporal classifiers not folded", mode, err)
		}
		idx := strings.Index(string(r.Document), `"/b":`)
		r.Document = append(append([]byte(nil), r.Document[:idx]...), []byte(strings.Replace(string(r.Document[idx:]), "2024-02-29T12:34:56+02:30", "2024-02-29T10:04:56Z", 1))...)
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "BetaResponseDto"), "typealias BetaResponseDto") {
			t.Fatal("source timestamp identity collapsed", mode, err)
		}
	}
}
