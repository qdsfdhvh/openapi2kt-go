package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestFormFieldDefaults(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/form-field-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.ModelStyle, rq.Options.Validation = style, checked
			for _, optimized := range []bool{false, true} {
				rq.Options.DeduplicateModels, rq.Options.HoistSharedShapes, rq.Options.MergeIdenticalModels = optimized, optimized, optimized
				rq.Options.LockHoistedNames = optimized
				report, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(style, checked, optimized, err)
				}
				api := source(t, report, "DefaultApi")
				for _, want := range []string{`input.text?.let { this.append("text", it) }`, `input.code?.let { this.append("code", it.serialName) }`, `append("requiredValue", input.requiredValue)`} {
					if !strings.Contains(api, want) {
						t.Fatal("default changed the actual form encoder", want)
					}
				}
				var declarations strings.Builder
				for _, f := range report.Artifacts.Files {
					declarations.WriteString(f.Content)
				}
				for _, want := range []string{`val requiredValue: kotlin.String,`, `val text: kotlin.String? = "a b+&雪"`, `val active: kotlin.Boolean? = false`, `val count: kotlin.Int? = 0`} {
					if !strings.Contains(declarations.String(), want) {
						t.Fatal("lost field-own initializer or required suppression", want)
					}
				}
				again, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(report, again) {
					t.Fatal("form defaults are not deterministic", err)
				}
			}
		}
	}
}

func TestFormFieldDefaultsValidateBeforeRequiredOrExcludedSuppression(t *testing.T) {
	for _, field := range []string{
		`{"type":"string","default":null}`,
		`{"type":"integer","default":2147483648}`,
		`{"type":"string","enum":["ready"],"default":"unknown"}`,
		`{"type":"integer","enum":[1,2],"default":3}`,
		`{"type":"string","format":"date","default":"2026-02-30"}`,
		`{"type":"string","format":"uuid","default":"invalid"}`,
		`{"type":"string","minLength":2,"default":"a"}`,
	} {
		for _, required := range []string{"", `,"required":["value"]`} {
			for _, excluded := range []bool{false, true} {
				rq := request(formRequestDocument(`{"type":"object","properties":{"value":`+field+`}`+required+`}`, true), "Dto")
				rq.Options.Validation = true
				if excluded {
					rq.Options.ExcludeAPIs = []string{"DefaultApi"}
				}
				report, err := core.GenerateWithDiagnostics(rq)
				if err == nil || len(report.Artifacts.Files) != 0 {
					t.Fatal("invalid annotation bypassed validation", field, required, excluded, err)
				}
				d := err.(core.Diagnostics)[0]
				if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || !strings.HasSuffix(*d.Pointer, "/properties/value/default") {
					t.Fatal("lost original default diagnostic", field, err)
				}
			}
		}
	}
}

func TestFormFieldDefaultsFormerRefusals(t *testing.T) {
	for _, schema := range []string{
		`{"type":"object","required":["v"],"properties":{"v":{"type":"string","default":"a"}}}`,
		`{"type":"object","properties":{"value":{"type":"string","default":"value"}}}`,
		`{"type":"object","properties":{"code":{"type":"string","enum":["a"],"default":"a"}}}`,
		`{"type":"object","properties":{"value":{"type":"number","default":1.25}}}`,
	} {
		if _, err := core.Generate(request(formRequestDocument(schema, true), "Dto")); err != nil {
			t.Fatal("former form default refusal remains", err)
		}
	}
}
