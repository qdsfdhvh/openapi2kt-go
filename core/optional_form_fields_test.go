package core_test

import (
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestOptionalFlatFormFields(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, required := range []bool{false, true} {
			rq := request(formRequestDocument(`{"type":"object","properties":{"text":{"type":"string"},"active":{"type":"boolean"},"count":{"type":"integer"},"large":{"type":"integer","format":"int64"}}}`, required), "Dto")
			rq.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("optional flat form fields rejected", checked, required, err)
			}
		}
	}
}

func TestOptionalFlatFormFieldSourceGuards(t *testing.T) {
	// Keep the exact former optional String vocabulary refusal as a positive.
	if _, err := core.GenerateWithDiagnostics(request(formRequestDocument(`{"type":"object","properties":{"value":{"type":"string","enum":["value"]}}}`, true), "Dto")); err != nil {
		t.Fatal("optional String enum form rejected", err)
	}
	for _, field := range []string{
		`{"type":["string","null"]}`,
		`{"type":"string","nullable":true}`,
		`{"type":"string","default":"value"}`,
		`{"type":"array","items":{"type":"string"}}`,
	} {
		rq := request(formRequestDocument(`{"type":"object","properties":{"value":`+field+`}}`, true), "Dto")
		if report, err := core.GenerateWithDiagnostics(rq); err == nil || len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_OPERATIONS_UNSUPPORTED" {
			t.Fatal("optional form field widened source grammar", field, err)
		}
	}
	// Explicit nullable request properties would introduce Patch states.
	input := strings.Replace(string(formRequestDocument(`{"type":"object","properties":{"value":{"type":"string","nullable":true}}}`, true)), `"get":`, `"patch":`, 1)
	if _, err := core.GenerateWithDiagnostics(request([]byte(input), "Dto")); err == nil {
		t.Fatal("optional field granted Patch form permission")
	}
}

func TestOptionalFlatFormFieldCheckedSourceAndDeterministicPlan(t *testing.T) {
	for _, checked := range []bool{false, true} {
		rq := request(formRequestDocument(`{"type":"object","properties":{"text":{"type":"string","minLength":2},"flag":{"type":"boolean"},"class":{"type":"integer","format":"int64"}}}`, false), "Dto")
		rq.Options.Validation = checked
		report, err := core.GenerateWithDiagnostics(rq)
		if !checked {
			if err == nil {
				t.Fatal("unchecked constraints acquired permission")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		api := source(t, report, "DefaultApi")
		for _, want := range []string{`input.text?.let { this.append("text", it) }`, `input.flag?.let { this.append("flag", it.toString()) }`, `input.` + "`class`" + `?.let { this.append("class", it.toString()) }`} {
			if !strings.Contains(api, want) {
				t.Fatal("optional field lost mapped omission", want, api)
			}
		}
	}
}
