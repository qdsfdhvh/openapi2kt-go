package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestFloatingFormFields(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, required := range []bool{false, true} {
			for _, fields := range []string{`"required":["floatValue","doubleValue"]`, `"required":[]`} {
				rq := request(formRequestDocument(`{"type":"object","properties":{"floatValue":{"type":"number","format":"float"},"doubleValue":{"type":"number"}},`+fields+`}`, required), "Dto")
				rq.Options.Validation = checked
				if _, err := core.GenerateWithDiagnostics(rq); err != nil {
					t.Fatal("floating form fields rejected", checked, required, fields, err)
				}
			}
		}
	}
}

func TestFloatingFormFieldSourceGuards(t *testing.T) {
	// Keep the exact former optional numeric vocabulary refusal as a positive.
	if _, err := core.GenerateWithDiagnostics(request(formRequestDocument(`{"type":"object","properties":{"value":{"type":"number","enum":[1.25]}}}`, true), "Dto")); err != nil {
		t.Fatal("optional floating enum form rejected", err)
	}
	for _, field := range []string{
		`{"type":["number","null"]}`,
		`{"type":"number","nullable":true}`,
		`{"type":"number","default":1.25}`,
		`{"type":"number","x-native-type":"kotlin.Double"}`,
	} {
		rq := request(formRequestDocument(`{"type":"object","properties":{"value":`+field+`}}`, true), "Dto")
		if report, err := core.GenerateWithDiagnostics(rq); err == nil || len(report.Artifacts.Files) != 0 {
			t.Fatal("floating form widened source grammar", field, err)
		}
	}
	for _, checked := range []bool{false, true} {
		rq := request(formRequestDocument(`{"type":"object","properties":{"value":{"type":"number","format":"float","minimum":0,"maximum":10}}}`, true), "Dto")
		rq.Options.Validation = checked
		_, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != checked {
			t.Fatal("floating constraints lost validation gate", err)
		}
	}
}

func TestFloatingFormConstructorSlots(t *testing.T) {
	for _, width := range []int{120, 121} {
		var fields, names []string
		for i := range width {
			name := fmt.Sprintf("p%d", i)
			fields = append(fields, `"`+name+`":{"type":"number"}`)
			names = append(names, `"`+name+`"`)
		}
		rq := request(formRequestDocument(`{"type":"object","properties":{`+strings.Join(fields, ",")+`},"required":[`+strings.Join(names, ",")+`]}`, true), "Dto")
		report, err := core.GenerateWithDiagnostics(rq)
		if width == 120 && err != nil {
			t.Fatal("JVM240 Double slots rejected", err)
		}
		if width == 121 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(report.Artifacts.Files) != 0) {
			t.Fatal("Double slots escaped240", err)
		}
	}
}
