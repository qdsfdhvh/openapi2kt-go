package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleSelectedReferenceOwnDefaults(t *testing.T) {
	for _, target := range []struct {
		schema, value string
		checked       bool
	}{
		{`{"type":"string"}`, `["ready"]`, false},
		{`{"type":"string","format":"uuid"}`, `["550e8400-e29b-41d4-a716-446655440000"]`, false},
		{`{"type":"string","format":"date"}`, `["2026-10-27"]`, false},
		{`{"type":"string","format":"date-time"}`, `["2026-10-27T12:30:00Z"]`, false},
		{`{"type":"integer","format":"int64"}`, `[9007199254740993]`, false},
		{`{"enum":["ready","later"]}`, `["ready"]`, false},
		{`{"type":"number","format":"float","enum":[0.1]}`, `[0.1]`, true},
		{`{"type":"boolean","enum":[true,false]}`, `[false]`, true},
		{`{"properties":{"name":{"type":"string"},"n":{"type":"integer","default":7}},"required":["name"]}`, `[{"name":"outer"}]`, false},
		{`{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer","default":7}}}]}`, `[{"name":"outer"}]`, false},
	} {
		for _, tail := range []string{"", `,"items":false`, `,"items":{"$ref":"#/components/schemas/Alias","nullable":false}`} {
			for _, checked := range []bool{false, true} {
				if target.checked && !checked {
					continue
				}
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					tuple := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias","nullable":false}]` + tail + `,"default":` + target.value + `}`
					rq := request(schemas(`{"Value":`+target.schema+`,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Rows":`+tuple+`,"Wrapped":{"$ref":"#/components/schemas/Rows","default":[]},"Holder":{"properties":{"rows":`+tuple+`}}}`), "Dto")
					rq.Options.Validation, rq.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatalf("%+v tail=%s checked=%t diagnostic=%+v", target, tail, checked, err.(core.Diagnostics))
					}
					if !strings.Contains(source(t, got, "RowsDto"), "RowsDto default") || !strings.Contains(source(t, got, "HolderDto"), "openapi-kt default") {
						t.Fatal("missing actual tuple own initializers")
					}
					again, e := core.GenerateWithDiagnostics(rq)
					if e != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("reference tuple default nondeterminism", e)
					}
				}
			}
		}
	}
}

func TestTupleReferenceDefaultInternalAndCountSources(t *testing.T) {
	for _, target := range []string{
		`{"type":"integer"}`,
		`{"enum":["ready"]}`,
		`{"properties":{"name":{"type":"string"}},"required":["name"]}`,
		`{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]}]}`,
	} {
		value := `[{"name":"outer"}]`
		if strings.Contains(target, `"enum"`) {
			value = `["ready"]`
		} else if strings.Contains(target, `"integer"`) {
			value = `[1]`
		}
		tuple := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Source/properties/value"}],"default":` + value + `}`
		rq := request(schemas(`{"Source":{"properties":{"value":`+target+`}},"Rows":`+tuple+`,"Holder":{"properties":{"rows":`+tuple+`}}}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.Generate(rq); err != nil {
			t.Fatal("internal selected default source", target, err)
		}
	}
	for _, target := range []string{
		`{"properties":{"name":{"type":"string"},"n":{"type":"integer","default":7}},"required":["name"],"minProperties":2}`,
		`{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer","default":7}}}],"minProperties":2}`,
	} {
		for _, value := range []string{`[{"name":"outer","n":1}]`, `[{"name":"outer"}]`} {
			rq := request(schemas(`{"Value":`+target+`,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"default":`+value+`}}`), "Dto")
			rq.Options.Validation = true
			_, err := core.Generate(rq)
			if strings.Contains(value, `"n"`) && err != nil || !strings.Contains(value, `"n"`) && (err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Rows/default/0") {
				t.Fatal("target object count omitted source supplied key check", target, value, err)
			}
		}
	}
}

func TestTupleReferenceDefaultLiteralBudgets(t *testing.T) {
	for _, count := range []int{16383, 16384} {
		values := strings.TrimSuffix(strings.Repeat("0,", count), ",")
		rq := request(schemas(`{"Value":{"type":"integer"},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"default":[`+values+`]}}`), "Dto")
		_, err := core.Generate(rq)
		if count == 16383 && err != nil || count == 16384 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("reference tuple value budget", count, err)
		}
	}
	for _, length := range []int{32764, 32765} {
		input := schemas(fmt.Sprintf(`{"Value":{"type":"string"},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"default":["%s"]}}}}`, strings.Repeat("a", length)))
		_, err := core.Generate(request(input, "Dto"))
		if length == 32764 && err != nil || length == 32765 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("reference tuple byte budget", length, err)
		}
	}
}

func TestTupleReferenceDefaultIndependentAnnotations(t *testing.T) {
	for _, target := range []struct{ schema, value string }{
		{`{"type":"string","default":"target"}`, `["consumer"]`},
		{`{"enum":["target","consumer"],"default":"target"}`, `["consumer"]`},
		{`{"properties":{"name":{"type":"string"}},"required":["name"],"default":{"name":"target"}}`, `[{"name":"consumer"}]`},
	} {
		tuple := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias","nullable":false}],"default":` + target.value + `}`
		input := schemas(`{"Value":` + target.schema + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Rows":` + tuple + `,"Holder":{"properties":{"rows":` + tuple + `,"borrowed":{"$ref":"#/components/schemas/Rows"}}}}`)
		if _, err := core.Generate(request(input, "Dto")); err != nil {
			t.Fatalf("%+v diagnostic=%+v", target, err.(core.Diagnostics))
		}
	}
}

func TestTupleReferenceDefaultNullableAndStrictValues(t *testing.T) {
	for _, root := range []bool{false, true} {
		for _, tc := range []struct{ target, value, pointer string }{
			{`{"enum":["ready"]}`, `["future"]`, "/default/0"},
			{`{"type":"integer","enum":[1,2]}`, `[3]`, "/default/0"},
			{`{"type":"string","format":"date"}`, `["2026-02-30"]`, "/default/0"},
			{`{"properties":{"name":{"type":"string"}},"required":["name"]}`, `[{}]`, "/default/0"},
			{`{"properties":{"name":{"type":"string"}}}`, `[{"extra":true}]`, "/default/0/extra"},
		} {
			tuple := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"default":` + tc.value + `}`
			declaration, pointer := `"Rows":`+tuple, "/components/schemas/Rows"
			if !root {
				declaration = `"Holder":{"properties":{"rows":` + tuple + `},"required":["rows"]}`
				pointer = "/components/schemas/Holder/properties/rows"
			}
			rq := request(schemas(`{"Value":`+tc.target+`,`+declaration+`}`), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, root, err)
			}
			if d := err.(core.Diagnostics)[0]; d.Pointer == nil || *d.Pointer != pointer+tc.pointer {
				t.Fatal(tc, root, err)
			}
		}
	}
	input := schemas(`{"Value":{"enum":["ready"]},"Nullable":{"$ref":"#/components/schemas/Value","nullable":true},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Nullable"}],"default":[null,"ready"]},"Narrow":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Nullable","nullable":false}],"default":["ready"]}}`)
	if _, err := core.Generate(request(input, "Dto")); err != nil {
		t.Fatal("nearest element nullable lost", err)
	}
	input = []byte(strings.Replace(string(input), `"nullable":false}],"default":["ready"]`, `"nullable":false}],"default":[null]`, 1))
	failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Narrow/default/0")
}
