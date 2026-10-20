package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableAllOfComponentNullFactory(t *testing.T) {
	for _, shape := range []string{
		// Preserve the exact input previously rejected by the null-factory guard.
		`"allOf":[{"properties":{"a":{"type":"string"}}}],"nullable":true`,
		`"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}],"nullable":true`,
		`"type":["object","null"],"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]}]`,
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				body := `{"N":{` + shape + `,"default":null},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/N"}}}}`
				r := request(schemas(body), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(shape, checked, style, err)
				}
				text := source(t, got, "NDto")
				if !strings.Contains(text, "public fun default(): NDto? = null") || strings.Contains(text, "openapi-kt component default") {
					t.Fatal("allOf null factory must return null without a decode helper", text)
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic factory", err)
				}
				r.Document = schemas(strings.Replace(body, `,"default":null`, "", 1))
				control, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal("unannotated allOf control", err)
				}
				if source(t, got, "HolderDto") != source(t, control, "HolderDto") || !reflect.DeepEqual(got.Warnings, control.Warnings) {
					t.Fatal("component default inherited or warnings changed")
				}
				if strings.Contains(source(t, control, "NDto"), "fun default()") {
					t.Fatal("unannotated component gained a factory")
				}
			}
		}
	}
}

func TestNullableAllOfComponentNullFactoryGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ name, body string }{
			{"non-nullable", `{"N":{"allOf":[{"properties":{"a":{"type":"string"}}}],"default":null}}`},
			{"explicit false", `{"N":{"allOf":[{"properties":{"a":{"type":"string"}}}],"nullable":false,"default":null}}`},
			{"root counts", `{"N":{"allOf":[{"properties":{"a":{"type":"string"}}}],"nullable":true,"minProperties":0,"default":null}}`},
			{"member counts", `{"N":{"allOf":[{"properties":{"a":{"type":"string"}},"minProperties":0}],"nullable":true,"default":null}}`},
			{"empty", `{"N":{"allOf":[{"type":"object"}],"nullable":true,"default":null}}`},
			{"discarded field", `{"N":{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"a":{"type":"integer"}}}],"nullable":true,"default":null}}`},
			{"unsupported omitted child", `{"N":{"allOf":[{"properties":{"a":{"type":"string","x-native-type":"org.example.Money"}}}],"nullable":true,"default":null}}`},
			{"invalid omitted child default", `{"N":{"allOf":[{"properties":{"a":{"type":"integer","default":"bad"}}}],"nullable":true,"default":null}}`},
			{"invalid discarded child", `{"N":{"allOf":[{"properties":{"a":{"type":"invalid"}}},{"properties":{"a":{"type":"string"}}}],"nullable":true,"default":null}}`},
			{"union product", `{"N":{"allOf":[{"oneOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"b":{"type":"integer"}}}]}],"nullable":true,"default":null}}`},
			{"explicit false wrapper", `{"T":{"allOf":[{"properties":{"a":{"type":"string"}}}]},"N":{"$ref":"#/components/schemas/T","nullable":false,"default":null}}`},
		} {
			t.Run(fmt.Sprintf("%s/checked=%t", tc.name, checked), func(t *testing.T) {
				r := request(schemas(tc.body), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Pointer == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("null annotation bypassed source validation", err, got)
				}
				if tc.name == "non-nullable" || tc.name == "explicit false" {
					if ds[0].Code != "E_SCHEMA_INVALID" || *ds[0].Pointer != "/components/schemas/N/default" {
						t.Fatal("null value lost its original diagnostic", err)
					}
				}
			})
		}
		// Null must not skip the source constructor-slot budget.
		var fields []string
		for i := 0; i < 241; i++ {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
		}
		r := request(schemas(`{"N":{"allOf":[{"properties":{`+strings.Join(fields, ",")+`}}],"nullable":true,"default":null}}`), "Dto")
		r.Options.Validation = checked
		_, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" {
			t.Fatal("null bypassed constructor budget", err)
		}
	}
}
