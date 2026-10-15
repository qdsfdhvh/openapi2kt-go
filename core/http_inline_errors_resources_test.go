package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestHTTPInlineErrorsDepthSlotsAndSelectedDeclarations(t *testing.T) {
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"p%d":{"type":"string"}`, i)
		}
		doc := httpDoc(`{"operationId":"fetch","responses":{"204":{},"400":{"content":{"application/json":{"schema":{"properties":{` + strings.Join(fields, ",") + `}}}}}}}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if (err == nil) != (count == 240) {
			t.Fatal(count, err)
		}
		if err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal(got, err)
		}
	}
	for _, depth := range []int{47, 48} {
		field := `{"type":"string"}`
		for i := range depth {
			field = fmt.Sprintf(`{"properties":{"n%d":%s},"required":["n%d"]}`, i, field, i)
		}
		doc := httpDoc(`{"operationId":"fetch","responses":{"204":{},"400":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}}}}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if (err == nil) != (depth == 47) {
			t.Fatal(depth, err)
		}
		if err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal(got, err)
		}
	}
	doc := httpDoc(`{"operationId":"fetch","responses":{"204":{},"400":{"content":{"application/problem+json":{"schema":{"properties":{"ignored":{"type":"string"}}}},"application/json":{"schema":{"properties":{"message":{"type":"string"}}}}}}}}`)
	got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	if text := source(t, got, "FetchError400Dto"); strings.Contains(text, "ignored") || !strings.Contains(text, "message") {
		t.Fatal("unselected media declaration leaked", text)
	}
	rq := request(doc, "Dto")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	excluded, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(excluded.Artifacts.Files) != 1 {
		t.Fatal("excluded errors retained DTO/helper", excluded, err)
	}
}
