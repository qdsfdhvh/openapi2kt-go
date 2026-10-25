package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestStandardFormFields(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, required := range []bool{false, true} {
			rq := request(formRequestDocument(`{"type":"object","properties":{"uuid":{"type":"string","format":"uuid"},"date":{"type":"string","format":"date"},"instant":{"type":"string","format":"date-time"}},"required":["uuid","date"]}`, required), "Dto")
			rq.Options.Validation = checked
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				t.Fatal("standard scalar form fields rejected", checked, required, e)
			}
		}
	}
}

func TestStandardFormFieldsFormerNegativesAndSourceGuards(t *testing.T) {
	// Retain the exact former UUID refusals as executable positive inputs.
	for _, s := range []string{`{"type":"object","properties":{"value":{"type":"string","format":"uuid"}}}`, `{"type":"object","required":["v"],"properties":{"v":{"type":"string","format":"uuid"}}}`} {
		if _, e := core.Generate(request(formRequestDocument(s, true), "Dto")); e != nil {
			t.Fatal(e)
		}
	}
	for _, format := range []string{"uuid", "date", "date-time"} {
		for _, addition := range []string{`,"nullable":true`, `,"default":null`, `,"enum":["2026-01-01"]`, `,"x-native-type":"caller.Custom"`, `,"readOnly":true`} {
			rq := request(formRequestDocument(`{"type":"object","properties":{"value":{"type":"string","format":"`+format+`"`+addition+`}}}`, true), "Dto")
			if report, e := core.GenerateWithDiagnostics(rq); e == nil || len(report.Artifacts.Files) != 0 {
				t.Fatal("form source grammar widened", format, addition, e)
			}
		}
	}
	for _, format := range []string{"date", "date-time"} {
		rq := request(formRequestDocument(`{"type":"object","properties":{"value":{"type":"string","format":"`+format+`","minLength":1}}}`, true), "Dto")
		rq.Options.Validation = true
		if report, e := core.GenerateWithDiagnostics(rq); e == nil || len(report.Artifacts.Files) != 0 {
			t.Fatal("temporal bounds acquired permission", e)
		}
	}
}

func TestStandardFormFieldsUUIDChecksAndMappedText(t *testing.T) {
	for _, checked := range []bool{false, true} {
		rq := request(formRequestDocument(`{"type":"object","properties":{"uuid":{"type":"string","format":"uuid","pattern":"^[a-f0-9-]+$"},"date":{"type":"string","format":"date"},"instant":{"type":"string","format":"date-time"}},"required":["uuid","date"]}`, false), "Dto")
		rq.Options.Validation = checked
		report, e := core.GenerateWithDiagnostics(rq)
		if !checked {
			if e == nil {
				t.Fatal("unchecked UUID checks accepted")
			}
			continue
		}
		if e != nil {
			t.Fatal(e)
		}
		for _, want := range []string{`append("uuid", input.uuid.toString())`, `append("date", input.date.toString())`, `input.instant?.let { this.append("instant", it.toString()) }`} {
			if !strings.Contains(source(t, report, "DefaultApi"), want) {
				t.Fatal(want)
			}
		}
		again, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !reflect.DeepEqual(report, again) {
			t.Fatal("non-deterministic standard forms", e)
		}
	}
}

func TestStandardFormFieldsConstructorWidth(t *testing.T) {
	for _, width := range []int{240, 241} {
		var fields []string
		for i := range width {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string","format":"date-time"}`, i))
		}
		report, e := core.GenerateWithDiagnostics(request(formRequestDocument(`{"type":"object","properties":{`+strings.Join(fields, ",")+`}}`, true), "Dto"))
		if width == 240 && e != nil {
			t.Fatal(e)
		}
		if width == 241 && (e == nil || len(report.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("width cap changed", e)
		}
	}
}
