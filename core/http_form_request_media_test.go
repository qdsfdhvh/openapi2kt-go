package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func formRequestDocument(schema string, required bool) []byte {
	return httpDoc(fmt.Sprintf(`{"operationId":"submit","requestBody":{"required":%t,"content":{"application/x-www-form-urlencoded":{"schema":%s}}},"responses":{"204":{}}}`, required, schema))
}

func TestHTTPFormRequestMedia(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/form-request-media.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.ModelStyle, rq.Options.Validation = style, checked
			rq.Options.HoistSharedShapes, rq.Options.DeduplicateModels, rq.Options.MergeIdenticalModels = true, true, true
			first, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			api := source(t, first, "DefaultApi")
			for _, want := range []string{"FormDataContent", "Parameters", `append("text", input.text)`, `append("active", input.active.toString())`, "input.`class`", "Application.Json"} {
				if !strings.Contains(api, want) {
					t.Fatal("missing form/JSON emission", want, api)
				}
			}
			var declarations strings.Builder
			for _, artifact := range first.Artifacts.Files {
				declarations.WriteString(artifact.Content)
			}
			if !strings.Contains(declarations.String(), "val large: kotlin.Long,") {
				t.Fatal("lost actual constructor after hoist/merge")
			}
			second, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(first, second) {
				t.Fatal("form nondeterminism", err)
			}
		}
	}
}

func TestHTTPFormRequestSelectedOnlyBoundary(t *testing.T) {
	pointer := "/paths/~1ping/get/requestBody/content/application~1x-www-form-urlencoded/schema"
	for _, schema := range []string{
		`true`, `{"type":"string"}`, `{"type":"object"}`,
		`{"type":"object","nullable":true,"required":["v"],"properties":{"v":{"type":"string"}}}`,
		`{"type":"object","required":["v"],"properties":{"v":{"type":["string","null"]}}}`,
		`{"type":"object","required":["v"],"properties":{"v":{"type":"string","default":"a"}}}`,
		`{"type":"object","required":["v"],"properties":{"v":{"type":"array","items":{"type":"string"}}}}`,
		`{"allOf":[{"properties":{"v":{"type":"string"}},"required":["v"]}]}`,
		`{"type":"object","required":["v"],"properties":{"v":{"type":"string","readOnly":true}}}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(formRequestDocument(schema, true), ""))
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("accepted unsupported selected form", schema, err)
		}
		ds, ok := err.(core.Diagnostics)
		if !ok || ds[0].Code != "E_OPERATIONS_UNSUPPORTED" || (ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, pointer)) {
			t.Fatal(schema, err)
		}
	}
	valid := `{"type":"object","required":["v"],"properties":{"v":{"type":"string"}}}`
	// Preserve the exact former String enum form refusal as a positive.
	if _, err := core.Generate(request(formRequestDocument(`{"type":"object","required":["v"],"properties":{"v":{"type":"string","enum":["a"]}}}`, true), "Dto")); err != nil {
		t.Fatal("ordinary String enum form property rejected", err)
	}
	// Retain the exact former number-field refusal as a floating positive.
	if _, err := core.Generate(request(formRequestDocument(`{"type":"object","required":["v"],"properties":{"v":{"type":"number"}}}`, true), "Dto")); err != nil {
		t.Fatal("ordinary Double form property rejected", err)
	}
	// Preserve the exact former optional-property refusal as a positive.
	if _, err := core.Generate(request(formRequestDocument(`{"type":"object","properties":{"v":{"type":"string"}}}`, true), "Dto")); err != nil {
		t.Fatal("optional form property rejected", err)
	}
	// Retain the exact former optional-body case as an explicit positive.
	if _, err := core.Generate(request(formRequestDocument(valid, false), "Dto")); err != nil {
		t.Fatal("optional form body rejected", err)
	}
	// Exact form spelling wins even if a case variant has an inadmissible form
	// shape. Unselected schemas still take the original shared validation path.
	input := httpDoc(`{"requestBody":{"required":true,"content":{"application/x-www-form-urlencoded":{"schema":` + valid + `},"APPLICATION/X-WWW-FORM-URLENCODED":{"schema":{"type":"integer"}}}},"responses":{"204":{}}}`)
	if _, err := core.Generate(request(input, "")); err != nil {
		t.Fatal(err)
	}
	bad := httpDoc(`{"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"string"}},"application/x-www-form-urlencoded":{"schema":{"type":"string","format":"unsupported"}}}},"responses":{"204":{}}}`)
	failure(t, bad, "E_SCHEMA_UNSUPPORTED", pointer+"/format")
}

func TestHTTPFormRequestConstructorSlots(t *testing.T) {
	for _, width := range []int{240, 241} {
		var fields, names []string
		for i := range width {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
			names = append(names, fmt.Sprintf(`"p%d"`, i))
		}
		input := formRequestDocument(`{"type":"object","required":[`+strings.Join(names, ",")+`],"properties":{`+strings.Join(fields, ",")+`}}`, true)
		got, err := core.GenerateWithDiagnostics(request(input, ""))
		if width == 240 {
			if err != nil || !strings.Contains(source(t, got, "SubmitInput"), "val p239:") {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("lost constructor cap", err)
		}
	}
}

func TestHTTPFormRequestValidationAndExcludedSources(t *testing.T) {
	input := formRequestDocument(`{"type":"object","required":["v"],"properties":{"v":{"type":"string","minLength":1}}}`, true)
	rq := request(input, "")
	if _, err := core.Generate(rq); err == nil {
		t.Fatal("form bypassed validation opt-in")
	}
	rq.Options.Validation = true
	if _, err := core.Generate(rq); err != nil {
		t.Fatal(err)
	}
	bad := formRequestDocument(`{"type":"object","required":["v"],"properties":{"v":{"type":"string","default":"a"}}}`, true)
	rq = request(bad, "")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	if _, err := core.Generate(rq); err == nil {
		t.Fatal("excluded form source skipped validation")
	}
}
