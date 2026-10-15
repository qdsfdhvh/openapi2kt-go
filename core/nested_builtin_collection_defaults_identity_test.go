package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nestedBuiltinDefaultOperations(field string) []byte {
	op := `{"get":{"operationId":"alpha","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}}}}}`
	return []byte(`{"openapi":"3.1.0","paths":{"/a":` + op + `,"/b":` + strings.Replace(op, `"alpha"`, `"beta"`, 1) + `}}`)
}

func TestNestedBuiltinCollectionDefaultsCanonicalIdentityAndRetainedChecks(t *testing.T) {
	field := `{"additionalProperties":{"additionalProperties":{"type":"number","format":"float"}},"default":{"z":{"z":null},"a":{"z":1.25,"a":1}}}`
	field = strings.Replace(field, `"format":"float"`, `"format":"float","nullable":true`, 1)
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			base := field
			if checked {
				base = strings.Replace(base, `"nullable":true`, `"nullable":true,"minimum":0`, 1)
			}
			r := request(nestedBuiltinDefaultOperations(base), "Dto")
			r.Options.Validation = checked
			r.Options.ModelStyle = style
			r.Options.HoistSharedShapes = true
			r.Options.LockHoistedNames = true
			r.Options.DeduplicateModels = true
			r.Options.MergeIdenticalModels = true
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || len(got.Artifacts.NameLock) == 0 {
				t.Fatal(err)
			}
			canonical := source(t, got, "ResponseDto")
			if !strings.Contains(canonical, "kotlin.collections.Map<kotlin.String, kotlin.collections.Map<kotlin.String, kotlin.Float?>>") || !strings.Contains(source(t, got, "AlphaResponseDto"), "typealias AlphaResponseDto = org.example.models.models.ResponseDto") {
				t.Fatal("classifier/default lost during options", canonical)
			}
			r.PreviousNameLock = got.Artifacts.NameLock
			r.Document = []byte(strings.ReplaceAll(string(r.Document), `{"z":{"z":null},"a":{"z":1.25,"a":1}}`, `{"a":{"a":1.000000001,"z":1.250},"z":{"z":null}}`))
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nested canonical order/mapped float identity drift", err)
			}
			r.Document = nestedBuiltinDefaultOperations(strings.Replace(base, `"z":1.25`, `"z":2`, 1))
			changed, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
				t.Fatal("effective default reused frozen identity", err)
			}
			if checked {
				r.Document = nestedBuiltinDefaultOperations(strings.Replace(base, `"minimum":0`, `"minimum":1`, 1))
				if _, err := core.GenerateWithDiagnostics(r); err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" {
					t.Fatal("deep check lost in frozen identity", err)
				}
			}
		}
	}
}

func TestNestedBuiltinCollectionDefaultsNumericAndUniquenessChecks(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{nestedDefaultChain(2, `{"type":"number","format":"float","exclusiveMinimum":1}`, `[[1.000000001]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"number","minimum":0}`, `[[0.`+strings.Repeat("0", 256)+`1]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"integer","multipleOf":2}`, `[[3]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{`{"additionalProperties":{"type":"array","items":{"type":"string"},"uniqueItems":true},"default":{"x":["a","\u0061"]}}`, "E_SCHEMA_INVALID", "/default/x/1"},
		{nestedDefaultChain(2, `{"type":"string","pattern":"^a+$"}`, `[["b"]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"string","pattern":"."}`, `[["😀"]]`), "E_SCHEMA_UNSUPPORTED", "/default/0/0"},
	} {
		r := request(fieldDoc(tc.field), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || len(got.Artifacts.Files) != 0 {
			t.Fatal(tc, err)
		}
	}
	r := request(fieldDoc(`{"additionalProperties":{"type":"array","nullable":true,"items":{"type":"string","nullable":true,"minLength":1},"uniqueItems":true},"default":{"a":["😀",null],"b":null,"empty":[]}}`), "Dto")
	r.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(r); err != nil {
		t.Fatal("null bypass or Unicode codepoint boundary", err)
	}
}

func TestNestedBuiltinCollectionDifferentDefaultsStayDistinctUnderOptimization(t *testing.T) {
	field := nestedDefaultChain(2, `{"type":"integer"}`, `[[1]]`)
	input := nestedBuiltinDefaultOperations(field)
	// Change only the second operation's effective initializer.
	index := strings.Index(string(input), `"/b":`)
	input = append(append([]byte(nil), input[:index]...), []byte(strings.Replace(string(input[index:]), `[[1]]`, `[[2]]`, 1))...)
	for _, mode := range []string{"dedup", "merge", "combined"} {
		r := request(input, "Dto")
		r.Options.DeduplicateModels = mode != "merge"
		r.Options.MergeIdenticalModels = mode != "dedup"
		r.Options.HoistSharedShapes = mode == "combined"
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(mode, err)
		}
		for _, name := range []string{"AlphaResponseDto", "BetaResponseDto"} {
			if text := source(t, got, name); strings.Contains(text, "typealias "+name) || !strings.Contains(text, "class "+name) {
				t.Fatal("Different defaults collapsed actual classifiers", mode, name, text)
			}
		}
	}
}

func FuzzNestedBuiltinCollectionDefaults(f *testing.F) {
	for _, value := range []string{`[]`, `[[1,null],null,[]]`, `[[1.0]]`, `[["quote\"$😀"]]`, `[{"a/~":true,"$ref":false}]`, `{"z":{"b":1},"a":{}}`, `{"a":[null,1]}`} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > 4096 || !json.Valid([]byte(value)) {
			return
		}
		for _, schema := range []string{
			`{"type":"array","items":{"type":"array","nullable":true,"items":{"type":"integer","nullable":true}}`,
			`{"type":"array","items":{"additionalProperties":{"type":"boolean","nullable":true}}`,
			`{"additionalProperties":{"type":"array","items":{"type":"string","nullable":true}}`,
			`{"additionalProperties":{"additionalProperties":{"type":"number","format":"float","nullable":true}}`,
		} {
			for _, checked := range []bool{false, true} {
				r := request(fieldDoc(schema+`,"default":`+value+`}`), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					if len(got.Artifacts.Files) != 0 {
						t.Fatal("partial failure output")
					}
					continue
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic tree", err)
				}
			}
		}
	})
}
