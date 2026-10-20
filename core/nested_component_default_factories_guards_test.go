package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedComponentDefaultFactoryValuePointers(t *testing.T) {
	for _, tc := range []struct{ leaf, value, tail string }{
		{`{"type":"integer"}`, `[1.0]`, "/0"},
		{`{"type":"integer"}`, `[-0]`, "/0"},
		{`{"type":"integer"}`, `[2147483648]`, "/0"},
		{`{"type":"boolean"}`, `["true"]`, "/0"},
		{`{"type":"string"}`, `[null]`, "/0"},
		{`{"type":"string","nullable":true}`, `null`, ""},
		{`{"type":"string","format":"date","nullable":true}`, `[null,"2023-02-30"]`, "/1"},
		{`{"type":"string","format":"byte","nullable":true}`, `[null,[128]]`, "/1/0"},
		{`{"type":"string","enum":["one"],"nullable":true}`, `[null,"future"]`, "/1"},
		{`{"type":"integer","enum":[1,2],"nullable":true}`, `[null,3]`, "/1"},
	} {
		for _, checked := range []bool{false, true} {
			for _, wrapped := range []bool{false, true} {
				shape := `"additionalProperties":{"type":"array","items":` + tc.leaf + `}`
				body := `"N":{` + shape + `,"default":{"a/~":` + tc.value + `}}`
				if wrapped {
					body = `"T":{` + shape + `},"N":{"$ref":"#/components/schemas/T","default":{"a/~":` + tc.value + `}}`
				}
				r := request(schemas(`{`+body+`}`), "Dto")
				r.Options.Validation = checked
				report, err := core.GenerateWithDiagnostics(r)
				ds, ok := err.(core.Diagnostics)
				want := "/components/schemas/N/default/a~1~0" + tc.tail
				if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != want || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("nested value lost original kind/Pointer", tc, checked, wrapped, want, err)
				}
			}
		}
	}
	// Source insertion order controls failure precedence, even though valid maps
	// are canonicalized by sorting their keys.
	r := request(schemas(`{"N":{"additionalProperties":{"additionalProperties":{"type":"integer"}},"default":{"z":{"z":false,"a":1},"a":false}}}`), "Dto")
	_, err := core.GenerateWithDiagnostics(r)
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/N/default/z/z" {
		t.Fatal("canonical sorting moved the first invalid source value", err)
	}
}

func TestNestedComponentDefaultFactorySourceGuards(t *testing.T) {
	for _, leaf := range []string{
		`{"$ref":"#/components/schemas/T"}`,
		`{"allOf":[{"type":"string"}]}`,
		`{"oneOf":[{"type":"string"},{"type":"integer"}]}`,
		`{"type":"string","default":"one"}`,
		`{"type":"string","x-native-type":"org.example.Money"}`,
		`{"type":"string","x-kotlin-value-class":true}`,
		`{"type":"string","const":"one"}`,
		`{"type":"string","enum":["one","one"]}`,
		`{"type":"string","enum":["one",1]}`,
		`{"type":"string","nullable":"yes"}`,
		`{"type":"object","properties":{"x":{"type":"string"}}}`,
		`{"additionalProperties":true}`,
	} {
		for _, annotation := range []string{`[]`, `null`} {
			for _, checked := range []bool{false, true} {
				r := request(schemas(`{"T":{"type":"string"},"N":{"type":"array","nullable":true,"items":{"type":"array","items":`+leaf+`},"default":`+annotation+`}}`), "Dto")
				r.Options.Validation = checked
				if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("empty/null factory bypassed its source grammar", leaf, annotation, checked, err)
				}
			}
		}
	}
	for _, inner := range []string{
		`{"type":"array","items":{"type":"string"},"default":[]}`,
		`{"type":"array","items":{"type":"string"},"prefixItems":[{"type":"string"}]}`,
	} {
		if _, err := core.GenerateWithDiagnostics(request(schemas(`{"N":{"type":"array","items":`+inner+`,"default":[]}}`), "Dto")); err == nil {
			t.Fatal("nested container default/tuple role leaked", inner)
		}
	}
}

func TestNestedComponentDefaultFactoryChecksAndBudgets(t *testing.T) {
	for _, tc := range []struct {
		edges, limit int
		leaf         string
		checked      bool
	}{
		{48, 48, `{"type":"integer"}`, false},
		{49, 48, `{"type":"integer"}`, false},
		{16, 16, `{"type":"integer","minimum":0}`, true},
		{17, 16, `{"type":"integer","minimum":0}`, true},
	} {
		r := request(schemas(`{"N":`+nestedDefaultChain(tc.edges, tc.leaf, `[]`)+`}`), "Dto")
		r.Options.Validation = tc.checked
		report, err := core.GenerateWithDiagnostics(r)
		if tc.edges <= tc.limit && err != nil || tc.edges > tc.limit && (err == nil || !reflect.DeepEqual(report, core.GenerationReport{})) {
			t.Fatal("nested factory bypassed source/checked depth", tc, err)
		}
	}
	for _, tc := range []struct{ shape, value, tail string }{
		{`"type":"array","items":{"type":"array","minItems":1,"items":{"type":"integer"}}`, `[[]]`, "/0"},
		{`"type":"array","items":{"additionalProperties":{"type":"integer"},"maxProperties":0}`, `[{"a":1}]`, "/0"},
		{`"additionalProperties":{"type":"array","items":{"type":"integer","minimum":1}}`, `{"a/~":[0]}`, "/a~1~0/0"},
		{`"type":"array","items":{"type":"array","items":{"type":"integer","nullable":true},"uniqueItems":true}`, `[[null,null]]`, "/0/1"},
	} {
		r := request(schemas(`{"N":{`+tc.shape+`,"default":`+tc.value+`}}`), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/N/default"+tc.tail || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("nested factory lost recursive checks", tc, err)
		}
		r.Options.Validation = false
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("unchecked factory silently dropped constraints", tc)
		}
	}
	for _, size := range []int{32762, 32763} {
		r := request(schemas(`{"N":`+nestedDefaultChain(2, `{"type":"string"}`, `[["`+strings.Repeat("a", size)+`"]]`)+`}`), "Dto")
		report, err := core.GenerateWithDiagnostics(r)
		if size == 32762 {
			if err != nil {
				t.Fatal("32 KiB boundary rejected", err)
			}
		} else if ds, ok := err.(core.Diagnostics); !ok || ds[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("32 KiB boundary bypassed", err)
		}
	}
	for _, count := range []int{16382, 16383} {
		value := `[[` + strings.Repeat("0,", count-1) + `0]]`
		r := request(schemas(`{"N":`+nestedDefaultChain(2, `{"type":"integer"}`, value)+`}`), "Dto")
		report, err := core.GenerateWithDiagnostics(r)
		// The root and inner list each consume a value node.
		if count == 16382 {
			if err != nil {
				t.Fatal("bounded value nodes rejected", err)
			}
		} else if ds, ok := err.(core.Diagnostics); !ok || ds[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("value-node budget bypassed or returned partial artifacts", err)
		}
	}
}
