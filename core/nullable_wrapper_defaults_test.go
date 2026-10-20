package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestExplicitNullableWrapperDefaultFactories(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, tc := range []struct{ name, shape, value string }{
				{"object", `"properties":{"z":{"type":"string"}},"required":["z"]`, `{"z":"given"}`},
				{"allof", `"allOf":[{"properties":{"z":{"type":"string"}},"required":["z"]}]`, `{"z":"given"}`},
				{"list", `"type":"array","items":{"additionalProperties":{"type":["integer","null"]}}`, `[{"z":1,"a":null}]`},
				{"map", `"additionalProperties":{"type":"string"}`, `{"z":"given"}`},
				{"enum", `"type":"string","enum":["a","b"]`, `"a"`},
				{"date", `"type":"string","format":"date"`, `"2024-02-29"`},
				{"bytes", `"type":"string","format":"byte"`, `[-128,127]`},
				{"raw", `"type":"object"`, `[1,null]`},
				{"open", `"type":"object","additionalProperties":true`, `{"nil":null}`},
				{"scalar", `"type":"string","minLength":1`, `"given"`},
				{"native", `"type":"integer","format":"int32","enum":[1,2]`, `1`},
			} {
				if !checked && (tc.name == "scalar" || tc.name == "native") {
					continue
				}
				for _, value := range []string{tc.value, `null`} {
					t.Run(tc.name+"/"+string(style)+"/"+value, func(t *testing.T) {
						body := `{"T":{` + tc.shape + `},"M":{"$ref":"#/components/schemas/T"},"N":{"$ref":"#/components/schemas/M","nullable":true,"default":` + value + `},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/N"}}}}`
						r := request(schemas(body), "Dto")
						r.Options.Validation, r.Options.ModelStyle = checked, style
						got, err := core.GenerateWithDiagnostics(r)
						if err != nil {
							t.Fatal(checked, err)
						}
						wrapper := source(t, got, "NDto")
						if !strings.Contains(wrapper, "public fun `NDto default`(): NDto =") || !strings.Contains(wrapper, "public typealias NDto =") || !strings.Contains(wrapper, "?") {
							t.Fatal("nullable alias factory missing", wrapper)
						}
						if value == "null" && !strings.Contains(wrapper, `("null")`) {
							t.Fatal(wrapper)
						}
						again, err := core.GenerateWithDiagnostics(r)
						if err != nil || !reflect.DeepEqual(got, again) {
							t.Fatal("generation drift", err)
						}
						r.Document = schemas(strings.Replace(body, `,"default":`+value, "", 1))
						control, err := core.GenerateWithDiagnostics(r)
						if err != nil {
							t.Fatal("no-default control", err)
						}
						for _, name := range []string{"TDto", "MDto", "HolderDto"} {
							if source(t, got, name) != source(t, control, name) {
								t.Fatal("annotation escaped outer alias", name)
							}
						}
						if !reflect.DeepEqual(got.Warnings, control.Warnings) {
							t.Fatal("warning drift", got.Warnings, control.Warnings)
						}
					})
				}
			}
		}
	}
}

func TestExplicitNullableWrapperDefaultGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ shape, value, tail string }{
			{`"properties":{"z":{"type":"string"}},"required":["z"]`, `{}`, ""},
			{`"type":"array","items":{"additionalProperties":{"type":"integer"}}`, `[{"a/~":false}]`, "/0/a~1~0"},
			{`"type":"string","enum":["a"]`, `"future"`, ""},
			{`"type":"string","format":"date"`, `"2023-02-30"`, ""},
			{`"type":"string","format":"byte"`, `[128]`, "/0"},
			{`"additionalProperties":true`, `[]`, ""},
		} {
			r := request(schemas(`{"T":{`+tc.shape+`},"M":{"$ref":"#/components/schemas/T"},"N":{"$ref":"#/components/schemas/M","nullable":true,"default":`+tc.value+`}}`), "Dto")
			r.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/N/default"+tc.tail || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("nullable wrapper lost original value diagnostic", checked, tc, err)
			}
		}
		for _, tc := range []struct{ target, middle, outer string }{
			{`"type":"object"`, `"default":null,`, `"nullable":true`},
			{`"type":"object","additionalProperties":true,"default":false`, ``, `"nullable":true`},
			{`"type":"object"`, ``, `"nullable":false`},
			{`"oneOf":[{"type":"string"},{"type":"integer"}]`, ``, `"nullable":true`},
			{`"type":"string","const":"a"`, ``, `"nullable":true`},
			{`"type":"string","x-kotlin-value-class":true`, ``, `"nullable":true`},
			{`"type":"array","items":{"type":"string","default":"a"}`, ``, `"nullable":true`},
			{`"type":"array","items":{"$ref":"#/components/schemas/T"}`, ``, `"nullable":true`},
		} {
			r := request(schemas(`{"N":{"$ref":"#/components/schemas/M",`+tc.outer+`,"default":null},"M":{`+tc.middle+`"$ref":"#/components/schemas/T"},"T":{`+tc.target+`}}`), "Dto")
			r.Options.Validation = checked
			if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("null bypassed excluded source role", checked, tc, err)
			}
		}
	}
	for _, depth := range []int{44, 45} {
		r := request(schemas(`{"T":`+aliasDepthTree(depth, false)+`,"M":{"$ref":"#/components/schemas/T"},"N":{"$ref":"#/components/schemas/M","nullable":true,"default":null},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/N"}}}}`), "Dto")
		report, err := core.GenerateWithDiagnostics(r)
		if depth == 44 {
			if err != nil {
				t.Fatal(err)
			}
		} else if ds, ok := err.(core.Diagnostics); !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("null erased cached consumer depth", err)
		}
	}
}

func TestExplicitNullableWrapperDefaultBudgets(t *testing.T) {
	for _, tc := range []struct{ shape, value, code string }{
		{`"type":"object"`, `"` + strings.Repeat("x", 32768) + `"`, "E_MODEL_LIMIT"},
		{`"type":"object"`, `[` + strings.Repeat("0,", 16384) + `0]`, "E_MODEL_LIMIT"},
		{`"type":"array","items":{"type":"string","default":"x"}`, `null`, "E_SCHEMA_UNSUPPORTED"},
	} {
		r := request(schemas(`{"T":{`+tc.shape+`},"N":{"$ref":"#/components/schemas/T","nullable":true,"default":`+tc.value+`}}`), "Dto")
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != tc.code || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("wrapper erased default budget/grammar", tc.code, err)
		}
	}
}
