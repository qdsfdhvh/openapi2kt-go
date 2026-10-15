package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumNullMemberDiagnosticsAndExclusions(t *testing.T) {
	for _, tc := range []struct{ leaf, code, tail string }{
		{`{"type":["integer","null"],"enum":[1,null,null]}`, "E_SCHEMA_INVALID", "enum/2"},
		{`{"type":["integer","null"],"enum":[null]}`, "E_SCHEMA_INVALID", "enum"},
		{`{"type":"integer","enum":[1,null]}`, "E_SCHEMA_INVALID", "enum/1"},
		{`{"type":["integer","null"],"enum":[null,1,"bad"]}`, "E_SCHEMA_INVALID", "enum/2"},
		{`{"type":["integer","null"],"enum":[null,1],"minimum":2}`, "E_SCHEMA_INVALID", "enum"},
		{`{"type":["number","null"],"format":"float","enum":[null,16777216,16777217],"exclusiveMinimum":16777216}`, "E_SCHEMA_INVALID", "enum"},
		{`{"type":["integer","null"],"enum":[null,1,2],"default":3}`, "E_SCHEMA_INVALID", "default"},
		{`{"type":["integer","null"],"enum":[null,1,2],"minimum":2,"default":1}`, "E_SCHEMA_INVALID", "default"},
		{`{"type":["number","null"],"enum":[null,1,2],"multipleOf":0.5}`, "E_SCHEMA_UNSUPPORTED", "multipleOf"},
		{`{"type":["string","null"],"enum":["a",null]}`, "E_SCHEMA_INVALID", "enum/1"},
	} {
		r := request(fieldDoc(tc.leaf), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if tc.leaf == `{"type":["string","null"],"enum":["a",null]}` {
			// String null vocabularies have their separate open-enum contract.
			if e != nil || !strings.Contains(source(t, got, "RecordDto"), "public data object A") || !strings.Contains(source(t, got, "RecordDto"), "Unrecognized") {
				t.Fatal(tc, e)
			}
			continue
		}
		if e == nil || e.(core.Diagnostics)[0].Code != tc.code || !strings.HasSuffix(*e.(core.Diagnostics)[0].Pointer, "/"+tc.tail) || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc, e)
		}
	}
	for _, annotation := range []string{`,"x-native-type":"bigint"`, `,"x-kotlin-value-class":true`, `,"const":1`} {
		r := request(schemas(`{"Code":{"type":["integer","null"],"enum":[1,null]`+annotation+`}}`), "Dto")
		r.Options.Validation = true
		if got, e := core.GenerateWithDiagnostics(r); e == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal(annotation, e)
		}
	}
	for _, leaf := range []string{`{"type":["integer","null"],"enum":[null,1,2],"default":null}`, `{"type":"boolean","nullable":true,"enum":[true,null],"default":null}`} {
		r := request(fieldDoc(leaf), "Dto")
		r.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(r); e != nil {
			t.Fatal(leaf, e)
		}
	}
}

func TestNativeEnumNullMembersCountOriginalVocabularyAndSourceWork(t *testing.T) {
	for _, n := range []int{239, 240} {
		values := []string{"null"}
		for i := range n {
			values = append(values, fmt.Sprint(i))
		}
		r := request(fieldDoc(`{"type":["integer","null"],"enum":[`+strings.Join(values, ",")+`]}`), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (n == 239) || e != nil && (e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal(n, e)
		}
	}
	for _, n := range []int{46, 47} {
		r := request(rootSpecialChain(n, `{"type":["integer","null"],"enum":[1,null,2]}`, fmt.Sprintf(`{"properties":{"value":{"$ref":"#/components/schemas/A%02d","nullable":false}}}`, n-1)), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (n == 46) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal(n, e)
		}
	}
	for _, n := range []int{16, 17} {
		leaf := `{"type":["integer","null"],"enum":[null,1,2]}`
		for range n {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		r := request(fieldDoc(leaf), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (n == 16) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal(n, e)
		}
	}
}
