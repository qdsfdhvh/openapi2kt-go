package core_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPBodyGuardsAndSlots(t *testing.T) {
	for _, tc := range []struct{ body, code, tail string }{
		{`null`, "E_SCHEMA_INVALID", ""},
		{`{"required":"yes","content":{}}`, "E_SCHEMA_INVALID", "/required"},
		{`{"content":{"application/json":{"schema":{"allOf":[{"type":"string"},{"type":"boolean"}]}}}}`, "E_SCHEMA_UNSUPPORTED", "/content/application~1json/schema/allOf/0/type"},
		{`{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Missing"}}}}`, "E_REF_MISSING", "/content/application~1json/schema"},
	} {
		failure(t, httpDoc(`{"requestBody":`+tc.body+`,"responses":{"204":{}}}`), tc.code, "/paths/~1ping/get/requestBody"+tc.tail)
	}
	for _, n := range []int{119, 120} {
		var params []string
		for i := 0; i < n; i++ {
			params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"type":"integer","format":"int64"}}`, i))
		}
		input := httpDoc(`{"parameters":[` + strings.Join(params, ",") + `],"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"integer","format":"int64"}}}},"responses":{"204":{}}}`)
		if n == 119 {
			report(t, input, "")
		} else {
			failure(t, input, "E_OPERATION_LIMIT", "/paths/~1ping/get/parameters")
		}
	}
	s := source(t, report(t, httpDoc(`{"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"integer","nullable":true}}}},"responses":{"204":{}}}`), ""), "DefaultApi")
	if !strings.Contains(s, "input: kotlin.Int?,") || strings.Contains(s, "input: kotlin.Int? = null") {
		t.Fatal(s)
	}
}

func TestHTTPSimpleBodies(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "bodies.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := source(t, report(t, input, "Dto"), "DefaultApi")
	for _, want := range []string{"inputQuery: kotlin.String? = null", "input: org.example.models.models.WidgetDto,", "input: org.example.models.models.WidgetDto? = null,", "input: kotlin.Int,", "input: kotlin.String,", "setBody(input)", "contentType(io.ktor.http.ContentType.Application.Json)"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %s", want, s)
		}
	}
	if strings.Contains(source(t, report(t, input, "Dto"), "WidgetDto"), "Patch<") {
		t.Fatal("shared component gained request-only semantics")
	}
}
