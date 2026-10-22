package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func floatingUniqueGraduation(t *testing.T, input []byte) {
	t.Helper()
	rq := request(input, "Dto")
	if _, err := core.GenerateWithDiagnostics(rq); err == nil {
		t.Fatal("missing floating uniqueness opt-in")
	}
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("original floating uniqueness input did not graduate", err)
	}
}

func TestFloatingUniqueSources(t *testing.T) {
	for _, leaf := range []string{`{"type":"number"}`, `{"type":"number","format":"float","nullable":true}`, `{"type":"number","format":"double","minimum":0}`, `{"type":"number","enum":[1,2]}`, `{"type":"number","format":"float","enum":[1,1.000000001,2]}`, `{"anyOf":[{"type":"number","format":"float"},{"type":"null"}]}`} {
		rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":`+leaf+`}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(leaf, err)
		}
		text := source(t, got, "RecordDto")
		if !strings.Contains(text, "`openapi-kt unique floating items`(value)") || !strings.Contains(text, "isFinite()") {
			t.Fatal("floating check missing", text)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("floating uniqueness drift", err)
		}
	}
	for _, terminal := range []string{`{"type":"number","format":"float"}`, `{"type":"number","enum":[1,2]}`} {
		for _, ref := range []string{"Value", "Alias", "Owner/properties/value", "Bridge"} {
			rq := request(schemas(`{"ARecord":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/`+ref+`","nullable":true}}}},"Value":`+terminal+`,"Alias":{"$ref":"#/components/schemas/Value"},"Owner":{"properties":{"value":`+terminal+`}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/value"}}`), "Dto")
			rq.Options.Validation = true
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("existing floating reference rejected", terminal, ref, err)
			}
		}
	}
}

func TestFloatingUniqueDefaultEquality(t *testing.T) {
	for _, tc := range []struct {
		leaf, values string
		duplicate    bool
	}{
		{`{"type":"number","format":"float"}`, `[1,1.000000001]`, true},
		{`{"type":"number","format":"float"}`, `[1,1.0000001192092896]`, false},
		{`{"type":"number"}`, `[1,1.0000000000000001]`, true},
		{`{"type":"number"}`, `[1,1.0000000000000002]`, false},
		{`{"type":"number"}`, `[-0.0,0]`, true},
		{`{"type":"number","format":"float","enum":[1,1.000000001,2]}`, `[1,1.000000001]`, true},
		{`{"type":"number","nullable":true}`, `[null,null]`, true},
		{`{"type":"number","nullable":true}`, `[null,0]`, false},
	} {
		for _, nested := range []bool{false, true} {
			shape := `{"type":"array","uniqueItems":true,"items":` + tc.leaf + `}`
			values := tc.values
			path := "/default/1"
			if nested {
				shape = `{"type":"array","items":` + shape + `}`
				values = `[` + values + `]`
				path = "/default/0/1"
			}
			shape = strings.TrimSuffix(shape, "}") + `,"default":` + values + `}`
			rq := request(fieldDoc(shape), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if !tc.duplicate {
				if err != nil {
					t.Fatal(tc, nested, err)
				}
				continue
			}
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+path {
				t.Fatal("floating default equality differs", tc, nested, err)
			}
		}
	}
}

func TestFloatingNativeEnumOriginalGraduations(t *testing.T) {
	for _, leaf := range []string{`{"type":"number","enum":[1,2]}`, `{"type":"number","format":"float","enum":[1,2]}`} {
		floatingUniqueGraduation(t, fieldDoc(`{"type":"array","uniqueItems":true,"items":`+leaf+`}`))
	}
}

func TestFloatingUniqueBudgets(t *testing.T) {
	enumUniqueCheckedBudgets(t, `{"type":"number","format":"float"}`)
	enumUniqueSourceCacheBudgets(t, `{"type":["number","null"],"format":"float"}`)
	enumUniqueSourceCacheBudgets(t, `{"type":["number","null"],"enum":[1]}`)
}

func TestFloatingUniqueSourceExclusionsAndNoop(t *testing.T) {
	for _, leaf := range []string{`{"type":"number","const":1}`, `{"type":"string","x-native-type":"kotlin.Double"}`, `{"type":"string","x-native-type":"kotlin.Float"}`, `{"type":"array","items":{"type":"number"}}`} {
		rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":`+leaf+`}`), "Dto")
		rq.Options.Validation = true
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("projection gained floating equality", leaf, err)
		}
	}
	aliasFailure(t, schemas(`{"Value":{"type":"number","x-kotlin-value-class":true},"Record":{"properties":{"v":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Value"}}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/v/uniqueItems")
	for _, leaf := range []string{`{"type":"number","format":"float"}`, `{"type":"number","enum":[1,2]}`} {
		shape := `{"type":"array","uniqueItems":true,"items":` + leaf + `}`
		for _, doc := range [][]byte{schemas(`{"Record":{"properties":{"v":{"type":"string"}},"additionalProperties":` + shape + `}}`), schemas(`{"Record":{"allOf":[{"properties":{"v":` + shape + `}}],"properties":{"v":{"type":"string"}}}}`)} {
			rq := request(doc, "Dto")
			rq.Options.Validation = true
			if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("floating checks erased", err)
			}
		}
		a, b := request(fieldDoc(strings.Replace(shape, `true`, `false`, 1)), "Dto"), request(fieldDoc(strings.Replace(shape, `,"uniqueItems":true`, "", 1)), "Dto")
		a.Options.Validation, b.Options.Validation = true, true
		first, e1 := core.GenerateWithDiagnostics(a)
		second, e2 := core.GenerateWithDiagnostics(b)
		if e1 != nil || e2 != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("false changed floating artifacts", e1, e2)
		}
	}
}

func TestFloatingUniqueOriginalDefaultRoles(t *testing.T) {
	terminal := `{"type":"array","uniqueItems":true,"items":{"type":"number","format":"float"}}`
	for _, tc := range []struct{ body, path string }{
		{`{"Value":` + strings.TrimSuffix(terminal, "}") + `,"default":[1,1.000000001]}}`, "/components/schemas/Value/default/1"},
		{`{"Value":` + terminal + `,"Alias":{"$ref":"#/components/schemas/Value","default":[1,1.000000001]}}`, "/components/schemas/Alias/default/1"},
		{`{"Value":` + terminal + `,"Record":{"properties":{"v":{"$ref":"#/components/schemas/Value","default":[1,1.000000001]}}}}`, "/components/schemas/Record/properties/v/default/1"},
	} {
		rq := request(schemas(tc.body), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != tc.path {
			t.Fatal("floating default role/pointer lost", tc, err)
		}
	}
}
