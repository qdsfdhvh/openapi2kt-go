package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestCheckedStringEnumComponentRoots(t *testing.T) {
	for _, metadata := range []string{"", `,"format":"date"`, `,"x-native-type":"org.example.UnusedProvider"`} {
		for _, nullable := range []string{"", `,"nullable":true`} {
			body := `{"Word":{"type":"string","enum":["one","two","x"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$","default":"one"META NULLABLE},"Alias":{"$ref":"#/components/schemas/Word","default":"two"},"Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"Use":{"properties":{"word":{"$ref":"#/components/schemas/Alias","default":"one"},"rows":{"$ref":"#/components/schemas/Rows","default":["two"]},"unique":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Word"},"default":["one","two"]}},"default":{"word":"two","rows":["one"]}}}`
			body = strings.NewReplacer("META", metadata, "NULLABLE", nullable).Replace(body)
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("original checked enum root/aliases/defaults rejected", err, *err.(core.Diagnostics)[0].Pointer)
			}
			if !strings.Contains(source(t, got, "WordDto"), "class Unrecognized") || !strings.Contains(source(t, got, "UseDto"), "serialName.let") {
				t.Fatal("root vocabulary or typed consumer checks missing")
			}
			if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("checked enum root nondeterminism", err)
			}
			for _, tc := range []struct{ from, to, pointer string }{
				{`"default":"one"`, `"default":"x"`, "/components/schemas/Word/default"},
				{`"default":"two"`, `"default":"future"`, "/components/schemas/Alias/default"},
				{`"rows":["one"]`, `"rows":["x"]`, "/components/schemas/Use/default/rows/0"},
			} {
				enumBoundsDiagnostic(t, schemas(strings.Replace(body, tc.from, tc.to, 1)), true, "E_SCHEMA_INVALID", tc.pointer)
			}
			rq.Options.Validation = false
			if report, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("checked root bypassed validation opt-in", err)
			}
		}
	}
}

func TestCheckedStringEnumRootSourceAndBudgets(t *testing.T) {
	for _, tc := range []struct{ leaf, code, pointer string }{
		{`{"enum":["one","one"],"minLength":1}`, "E_SCHEMA_INVALID", "/enum/1"},
		{`{"enum":["one"],"minLength":-1}`, "E_SCHEMA_INVALID", "/minLength"},
		{`{"enum":["one"],"minLength":3,"maxLength":2}`, "E_SCHEMA_INVALID", "/maxLength"},
		{`{"enum":["one"],"pattern":"(?i)o"}`, "E_SCHEMA_UNSUPPORTED", "/pattern"},
		{`{"enum":["one"],"minimum":1}`, "E_SCHEMA_UNSUPPORTED", "/minimum"},
	} {
		body := `{"Word":` + tc.leaf + `,"Alias":{"$ref":"#/components/schemas/Word","nullable":true,"default":null}}`
		enumBoundsDiagnostic(t, schemas(body), true, tc.code, "/components/schemas/Word"+tc.pointer)
	}
	for _, hops := range []int{48, 49} {
		parts := []string{`"Word":{"enum":["one","x"],"minLength":2}`}
		for i := 0; i < hops; i++ {
			target := fmt.Sprintf("A%02d", i+1)
			if i == hops-1 {
				target = "Word"
			}
			annotation := ""
			if i == hops-1 {
				annotation = `,"default":"one"`
			}
			parts = append(parts, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"%s}`, i, target, annotation))
		}
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if hops == 48 && err != nil {
			t.Fatal("checked root rejected original48 alias hops", err)
		}
		if hops == 49 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("checked root exceeded49 alias hops", err)
		}
	}
	for _, size := range []int{32700, 32768} {
		word := strings.Repeat("a", size)
		rq := request(schemas(`{"Word":{"enum":["`+word+`"],"minLength":1},"Alias":{"$ref":"#/components/schemas/Word","default":"`+word+`"}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if size == 32700 && err != nil {
			t.Fatal("checked root valid default below32KiB rejected", err)
		}
		if size == 32768 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("checked root default escaped32KiB", err)
		}
	}
}

func checkedStringEnumRootGraduation(t *testing.T, rq core.GenerationRequest) {
	t.Helper()
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("original checked component-root refusal did not graduate", err)
	}
}

func TestCheckedStringEnumRootWithoutConsumers(t *testing.T) {
	for _, constraint := range []string{`"minLength":0`, `"pattern":".*"`, `"minLength":2,"maxLength":6`} {
		rq := request(schemas(`{"Word":{"enum":["one","x"],`+constraint+`}}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("root checks incorrectly require a consuming getter", err)
		}
	}
}

func TestCheckedStringEnumRootFormerRefusals(t *testing.T) {
	for _, body := range []string{
		`{"Leaf":{"enum":["one"],"minLength":2},"Use":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}}}}}}`,
		`{"Code":{"enum":["one"],"minLength":2},"Use":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Code"}}}}}`,
		`{"Value":{"enum":["one"],"minLength":2}}`,
		`{"Word":{"enum":["one","x"],"minLength":2}}`,
		`{"Word":{"enum":["one","x"],"minLength":2},"Alias":{"$ref":"#/components/schemas/Word"}}`,
	} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("original checked root refusal did not graduate", err)
		}
		rq.Options.Validation = false
		if _, err := core.GenerateWithDiagnostics(rq); err == nil {
			t.Fatal("old unchecked source refusal changed")
		}
	}
}
