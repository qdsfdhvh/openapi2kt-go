package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestEnumFormFields(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, required := range []bool{false, true} {
			rq := request(formRequestDocument(`{"type":"object","properties":{"code":{"type":"string","enum":["ready","done"]},"count":{"type":"integer","enum":[1,2]},"flag":{"type":"boolean","enum":[true]}},"required":["code"]}`, required), "Dto")
			rq.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("enum form fields rejected", checked, required, err)
			}
		}
	}
}

func TestEnumFormFieldSourceGuards(t *testing.T) {
	for _, field := range []string{
		`{"type":["string","null"],"enum":["a"]}`,
		`{"type":"string","enum":["a"],"default":"a"}`,
		`{"type":"string","format":"date","enum":["2026-01-01"]}`,
		`{"type":"string","enum":["a"],"x-native-type":"kotlin.String"}`,
		`{"type":"array","items":{"type":"string","enum":["a"]}}`,
	} {
		rq := request(formRequestDocument(`{"type":"object","properties":{"code":`+field+`}}`, true), "Dto")
		if report, err := core.GenerateWithDiagnostics(rq); err == nil || len(report.Artifacts.Files) != 0 {
			t.Fatal("enum form widened source grammar", field, err)
		}
	}
	for _, checked := range []bool{false, true} {
		rq := request(formRequestDocument(`{"type":"object","properties":{"code":{"type":"string","enum":["a"]},"optionalCode":{"type":"string","enum":["b"]}},"required":["code"]}`, true), "Dto")
		rq.Options.Validation = checked
		report, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		api := source(t, report, "DefaultApi")
		for _, want := range []string{`this.append("code", input.code.serialName)`, `input.optionalCode?.let { this.append("optionalCode", it.serialName) }`} {
			if !strings.Contains(api, want) {
				t.Fatal("form lost original enum wire names", want, api)
			}
		}
	}
}

func TestEnumFormOriginalVocabularyLimit(t *testing.T) {
	for _, native := range []bool{false, true} {
		for _, count := range []int{240, 241} {
			for _, checked := range []bool{false, true} {
				values := []any{}
				kind := "string"
				for i := range count {
					if native {
						kind = "integer"
						values = append(values, i)
					} else {
						values = append(values, "member"+strings.Repeat("x", i))
					}
				}
				field := map[string]any{"type": kind, "enum": values}
				schema := map[string]any{"type": "object", "properties": map[string]any{"code": field}}
				data, e := json.Marshal(schema)
				if e != nil {
					t.Fatal(e)
				}
				rq := request(formRequestDocument(string(data), true), "Dto")
				rq.Options.Validation = checked
				report, err := core.GenerateWithDiagnostics(rq)
				if count == 240 && err != nil {
					t.Fatal("vocabulary240 rejected", native, checked, err)
				}
				code := "E_ENUM_LIMIT"
				if native {
					code = "E_MODEL_LIMIT"
				}
				if count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != code || len(report.Artifacts.Files) != 0) {
					t.Fatal("enum form bypassed raw240", native, checked, err)
				}
			}
		}
	}
}
