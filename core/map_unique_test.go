package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func mapUniqueGraduation(t *testing.T, input []byte) {
	t.Helper()
	rq := request(input, "Dto")
	if _, err := core.GenerateWithDiagnostics(rq); err == nil {
		t.Fatal("Map uniqueness requires validation")
	}
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("original Map uniqueness input did not graduate", err)
	}
}

func TestMapUniqueExclusionsAndNoDrop(t *testing.T) {
	for _, leaf := range []string{
		`{"type":"number"}`, `{"type":"string","format":"byte"}`, `{"enum":["x"]}`,
		`{"type":"string","const":"x"}`, `{"type":"string","x-native-type":"kotlin.String"}`,
		`{"type":"array","items":{"type":"string"}}`, `{"additionalProperties":{"type":"string"}}`,
		`{"$ref":"#/components/schemas/Value"}`,
	} {
		rq := request(schemas(`{"Value":{"type":"string"},"Holder":{"properties":{"v":{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":`+leaf+`}}}}}`), "Dto")
		rq.Options.Validation = true
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("excluded Map value acquired equality", leaf, err)
		}
	}
	shape := `{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":{"type":"string"}}}`
	aliasFailure(t, schemas(`{"Holder":{"properties":{"v":{"type":"string"}},"additionalProperties":`+shape+`}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/additionalProperties/uniqueItems")
	aliasFailure(t, schemas(`{"Holder":{"allOf":[{"properties":{"v":`+shape+`}}],"properties":{"v":{"type":"string"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/allOf/0/properties/v/uniqueItems")
	a, b := request(fieldDoc(strings.Replace(shape, `true`, `false`, 1)), "Dto"), request(fieldDoc(strings.Replace(shape, `,"uniqueItems":true`, "", 1)), "Dto")
	a.Options.Validation, b.Options.Validation = true, true
	first, e1 := core.GenerateWithDiagnostics(a)
	second, e2 := core.GenerateWithDiagnostics(b)
	if e1 != nil || e2 != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("Map false output drift", e1, e2)
	}
}

func TestMapUniqueBudgets(t *testing.T) {
	base := `{"type":"array","uniqueItems":true,"items":{"type":"object","minProperties":0,"additionalProperties":{"type":"string","minLength":0}}}`
	for _, layers := range []int{15, 16} {
		shape := base
		for i := 1; i < layers; i++ {
			shape = `{"type":"array","items":` + shape + `}`
		}
		rq := request(fieldDoc(shape), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (layers == 15) || err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal("Map checked edge budget lost", layers, err)
		}
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"p%d":%s`, i, base)
		}
		rq := request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (width == 240) || err != nil && (err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("Map slot budget lost", width, err)
		}
		if err == nil && strings.Count(source(t, got, "HolderDto"), "private fun <T> `openapi-kt unique map items`") != 1 {
			t.Fatal("Map helper duplicated")
		}
	}
}

func TestMapUniqueDefaultRolesAndConstraints(t *testing.T) {
	shape := `{"type":"array","uniqueItems":true,"items":{"type":"object","minProperties":1,"additionalProperties":{"type":"string","minLength":1}}}`
	for _, tc := range []struct{ value, suffix string }{
		{`[{"a":"x","b":"y"},{"b":"y","a":"x"}]`, "/1"},
		{`[{}]`, "/0"}, {`[{"a":""}]`, "/0/a"},
	} {
		for _, role := range []string{"root", "alias", "field"} {
			body, pointer := "", ""
			switch role {
			case "root":
				body = `{"Rows":` + strings.TrimSuffix(shape, "}") + `,"default":` + tc.value + `}}`
				pointer = "/components/schemas/Rows/default"
			case "alias":
				body = `{"Rows":` + shape + `,"Alias":{"$ref":"#/components/schemas/Rows","default":` + tc.value + `}}`
				pointer = "/components/schemas/Alias/default"
			case "field":
				body = `{"Rows":` + shape + `,"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows","default":` + tc.value + `}}}}`
				pointer = "/components/schemas/Holder/properties/rows/default"
			}
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != pointer+tc.suffix {
				t.Fatal("Map default role/check/source lost", role, tc, err)
			}
		}
	}
}

func TestMapUniqueDirectSources(t *testing.T) {
	for _, value := range []string{`{"type":"string"}`, `{"type":"integer"}`, `{"type":"integer","format":"int64"}`, `{"type":"boolean"}`, `{"type":"string","nullable":true,"minLength":1}`} {
		field := `{"type":"array","uniqueItems":true,"items":{"type":"object","nullable":true,"minProperties":0,"additionalProperties":` + value + `}}`
		rq := request(fieldDoc(field), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(value, err)
		}
		if !strings.Contains(source(t, got, "RecordDto"), "`openapi-kt unique map items`(value)") {
			t.Fatal("Map check lost")
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Map source/output drift", err)
		}
	}
}

func TestMapUniqueDefaults(t *testing.T) {
	for _, tc := range []struct {
		leaf, values string
		valid        bool
	}{
		{`{"type":"string","nullable":true}`, `[{"a":"x","b":null},{"b":null,"a":"x"}]`, false},
		{`{"type":"integer"}`, `[{"a":0},{"a":0}]`, false},
		{`{"type":"integer","format":"int64"}`, `[{"a":9223372036854775807},{"a":9223372036854775807}]`, false},
		{`{"type":"boolean"}`, `[{"a":true},{"a":true}]`, false},
		{`{"type":"string","nullable":true}`, `[{},{"a":null},{"a":"null"},{"b":null},null]`, true},
		{`{"type":"string"}`, `[null,null]`, false},
	} {
		field := `{"type":"array","uniqueItems":true,"items":{"type":"object","nullable":true,"additionalProperties":` + tc.leaf + `}}`
		for _, nested := range []bool{false, true} {
			shape, annotation, suffix := field, tc.values, "/default/1"
			if nested {
				shape = `{"type":"array","items":` + field + `}`
				annotation = `[` + tc.values + `]`
				suffix = "/default/0/1"
			}
			shape = strings.TrimSuffix(shape, "}") + `,"default":` + annotation + `}`
			rq := request(fieldDoc(shape), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if tc.valid {
				if err != nil {
					t.Fatal(tc, nested, err)
				}
				continue
			}
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+suffix {
				t.Fatal("Map default equality/Pointer lost", tc, nested, err)
			}
		}
	}
}
