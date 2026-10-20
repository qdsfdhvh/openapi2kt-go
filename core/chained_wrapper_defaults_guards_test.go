package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestChainedWrapperDefaultValuePointersAndExclusions(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ target, value, tail string }{
			{`{"properties":{"z":{"type":"string"}},"required":["z"]}`, `{}`, ""},
			{`{"type":"array","items":{"additionalProperties":{"type":"integer"}}}`, `[{"a/~":false}]`, "/0/a~1~0"},
			{`{"type":"string","enum":["a"]}`, `"future"`, ""},
			{`{"type":"string","format":"date"}`, `"2023-02-30"`, ""},
			{`{"type":"string","format":"byte"}`, `[128]`, "/0"},
			{`{"type":"object","additionalProperties":true}`, `[]`, ""},
		} {
			r := request(schemas(`{"T":`+tc.target+`,"M":{"$ref":"#/components/schemas/T"},"N":{"$ref":"#/components/schemas/M","default":`+tc.value+`}}`), "Dto")
			r.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/N/default"+tc.tail || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("chain value lost outer annotation Pointer", tc, checked, err)
			}
		}
		for _, middle := range []string{
			`"$ref":"#/components/schemas/T","nullable":true`,
			`"$ref":"#/components/schemas/T","nullable":false`,
			`"$ref":"#/components/schemas/T","default":{"z":"middle"}`,
		} {
			r := request(schemas(`{"T":{"properties":{"z":{"type":"string"}}},"M":{`+middle+`},"N":{"$ref":"#/components/schemas/M","default":{"z":"outer"}}}`), "Dto")
			r.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_UNSUPPORTED" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/N/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("chain discarded an intermediate annotation", middle, checked, err)
			}
		}
		for _, body := range []string{
			`{"N":{"$ref":"#/components/schemas/N","default":{}}}`,
			`{"N":{"$ref":"#/components/schemas/M","default":{}},"M":{"$ref":"#/components/schemas/B"},"B":{"$ref":"#/components/schemas/M"}}`,
			`{"N":{"$ref":"#/components/schemas/M","default":{}},"M":{"$ref":"#/components/schemas/Missing"}}`,
			`{"N":{"$ref":"#/components/schemas/M","default":{}},"M":{"$ref":false}}`,
			`{"N":{"$ref":"#/components/schemas/M","default":{}},"M":{"$ref":"#/components/schemas/T","minLength":0},"T":{"type":"object"}}`,
			`{"N":{"$ref":"#/components/schemas/M","default":{}},"M":{"$ref":"#/components/schemas/T"},"T":{"type":"object","default":{}}}`,
			`{"N":{"$ref":"#/components/schemas/M","default":null},"M":{"$ref":"#/components/schemas/T"},"T":{"type":"object"}}`,
		} {
			r := request(schemas(body), "Dto")
			r.Options.Validation = checked
			if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("unsupported/cyclic chain bypassed validation", body, checked, err)
			}
		}
	}
}

func TestChainedWrapperDefaultScalarPolicies(t *testing.T) {
	for _, tc := range []struct{ shape, value string }{
		{`"type":"string"`, `"value"`},
		{`"type":"integer"`, `7`},
		{`"type":"number"`, `1.5`},
		{`"type":"boolean"`, `true`},
		{`"type":"integer","format":"int32","enum":[1,2]`, `2`},
		{`"type":"integer","format":"int64","enum":[1,2],"nullable":true`, `null`},
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(`{"T":{`+tc.shape+`},"M":{"$ref":"#/components/schemas/T"},"N":{"$ref":"#/components/schemas/M","default":`+tc.value+`}}`), "Dto")
			r.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(r)
			r.Document = []byte(strings.Replace(string(r.Document), `"N":{"$ref":"#/components/schemas/M"`, `"N":{"$ref":"#/components/schemas/T"`, 1))
			direct, directErr := core.GenerateWithDiagnostics(r)
			if (err == nil) != (directErr == nil) || !reflect.DeepEqual(report.Warnings, direct.Warnings) {
				t.Fatal("chain changed the direct wrapper's scalar policy", tc, checked, err, directErr)
			}
			if err != nil {
				if !reflect.DeepEqual(err, directErr) || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("chain changed scalar rejection diagnostics", tc, checked, err, directErr)
				}
			} else if !strings.Contains(source(t, report, "NDto"), "public fun `NDto default`()") {
				t.Fatal("admitted scalar chain lost factory", tc, checked)
			}
			if !strings.Contains(tc.shape, `"enum"`) && ((err == nil) != checked) {
				t.Fatal("plain scalar chain changed validation opt-in", tc, checked, err)
			}
		}
	}
}

func TestChainedWrapperDefaultPreservesCachedConsumerDepth(t *testing.T) {
	for _, depth := range []int{44, 45} {
		r := request(schemas(`{"T":`+aliasDepthTree(depth, false)+`,"M":{"$ref":"#/components/schemas/T"},"N":{"$ref":"#/components/schemas/M","default":[]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/N"}}}}`), "Dto")
		report, err := core.GenerateWithDiagnostics(r)
		if depth == 44 {
			if err != nil {
				t.Fatal("bounded cached chain rejected", err)
			}
		} else if ds, ok := err.(core.Diagnostics); !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("factory erased alias/consumer height", err)
		}
	}
}
