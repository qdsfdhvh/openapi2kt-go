package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRawJSONWrapperDefaultFactories(t *testing.T) {
	for _, target := range []string{
		`"properties":{}`,
		`"type":"object"`,
		`"type":"object","properties":{}`,
		`"type":"object","additionalProperties":true`,
		`"type":"object","nullable":true`,
		`"type":["object","null"],"properties":{}`,
		`"type":["object","null"],"additionalProperties":true`,
	} {
		values := []string{`{"z":{"/~":null,"$ref":"data"},"a":[1,true]}`}
		if !strings.Contains(target, "additionalProperties") {
			values = append(values, `[1,"two",false]`, `"null"`, `7`, `false`)
		}
		if strings.Contains(target, "null") {
			values = append(values, `null`)
		}
		for _, value := range values {
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					r := request(schemas(`{"T":{`+target+`},"N":{"$ref":"#/components/schemas/T","default":`+value+`},"Holder":{"required":["required"],"properties":{"optional":{"$ref":"#/components/schemas/N"},"required":{"$ref":"#/components/schemas/N"}}}}`), "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(target, value, checked, style, err)
					}
					text := source(t, got, "NDto")
					if !strings.Contains(text, "public fun `NDto default`(): NDto =") || !strings.Contains(text, "public typealias NDto = org.example.models.models.TDto") || !strings.Contains(text, "serializer<NDto>()") {
						t.Fatal("raw wrapper lost alias/helper", text)
					}
					again, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("raw wrapper output drifted", err)
					}
					r.Document = []byte(strings.Replace(string(r.Document), `,"default":`+value, "", 1))
					control, err := core.GenerateWithDiagnostics(r)
					if err != nil || source(t, got, "TDto") != source(t, control, "TDto") || source(t, got, "HolderDto") != source(t, control, "HolderDto") || !reflect.DeepEqual(got.Warnings, control.Warnings) {
						t.Fatal("raw wrapper changed its target/consumer/warnings", err)
					}
					if strings.Contains(source(t, control, "NDto"), "fun `NDto default`") {
						t.Fatal("unannotated raw wrapper gained a factory")
					}
				}
			}
		}
	}
}

func TestRawJSONWrapperDefaultGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ target, wrapper, code string }{
			{`"type":"object","additionalProperties":true`, `"default":[1]`, "E_SCHEMA_INVALID"},
			{`"type":"object","additionalProperties":true`, `"default":"null"`, "E_SCHEMA_INVALID"},
			{`"type":"object","additionalProperties":true,"nullable":true`, `"default":false`, "E_SCHEMA_INVALID"},
			{`"type":"object","additionalProperties":true`, `"default":null`, "E_SCHEMA_UNSUPPORTED"},
			{`"type":"object"`, `"default":null`, "E_SCHEMA_UNSUPPORTED"},
			{`"type":"object","nullable":false`, `"default":null`, "E_SCHEMA_UNSUPPORTED"},
			{`"type":"object","nullable":true`, `"nullable":false,"default":null`, "E_SCHEMA_UNSUPPORTED"},
			{`"type":"object","additionalProperties":true,"default":false`, `"default":{}`, "E_SCHEMA_INVALID"},
			{`"type":"object","format":"raw"`, `"default":{}`, "E_SCHEMA_UNSUPPORTED"},
		} {
			r := request(schemas(`{"T":{`+tc.target+`},"N":{"$ref":"#/components/schemas/T",`+tc.wrapper+`}}`), "Dto")
			r.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			pointer := "/components/schemas/N/default"
			if strings.Contains(tc.target, `"default":false`) {
				pointer = "/components/schemas/T/default"
			}
			if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("raw wrapper grammar or Pointer changed", tc, checked, err)
			}
		}
		for _, target := range []string{
			``,
			`"type":"object","properties":[]`,
			`"type":"object","required":true`,
			`"type":"object","nullable":"yes"`,
			`"type":"object","additionalProperties":true,"properties":{"a":{"type":"string"}}`,
			`"type":"object","nullable":true,"minProperties":0`,
		} {
			r := request(schemas(`{"T":{`+target+`},"N":{"$ref":"#/components/schemas/T","default":null}}`), "Dto")
			r.Options.Validation = checked
			if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("null raw wrapper bypassed original target validation", target, checked, err)
			}
		}
		r := request(schemas(`{"T":{"type":"object"},"M":{"$ref":"#/components/schemas/T","nullable":false,"default":null},"N":{"$ref":"#/components/schemas/M","default":{}}}`), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("chain ignored an invalid intermediate default")
		}
	}
}

func TestRawJSONWrapperDefaultBudgets(t *testing.T) {
	for _, target := range []string{"T", "M"} {
		for _, size := range []int{32766, 32767} {
			r := request(schemas(`{"T":{"type":"object"},"M":{"$ref":"#/components/schemas/T"},"N":{"$ref":"#/components/schemas/`+target+`","default":"`+strings.Repeat("a", size)+`"}}`), "Dto")
			report, err := core.GenerateWithDiagnostics(r)
			if size == 32766 {
				if err != nil {
					t.Fatal("raw wrapper rejected the 32 KiB boundary", err)
				}
			} else if ds, ok := err.(core.Diagnostics); !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/N/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("raw wrapper bypassed original byte budget/Pointer", err)
			}
		}
		// The root array consumes one node; each zero consumes one more.
		for _, count := range []int{16383, 16384} {
			value := `[` + strings.Repeat("0,", count-1) + `0]`
			r := request(schemas(`{"T":{"type":"object"},"M":{"$ref":"#/components/schemas/T"},"N":{"$ref":"#/components/schemas/`+target+`","default":`+value+`}}`), "Dto")
			report, err := core.GenerateWithDiagnostics(r)
			if count == 16383 {
				if err != nil {
					t.Fatal("raw wrapper rejected bounded nodes", err)
				}
			} else if ds, ok := err.(core.Diagnostics); !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/N/default/16383" || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("raw wrapper bypassed node budget/Pointer", err)
			}
		}
	}
}
