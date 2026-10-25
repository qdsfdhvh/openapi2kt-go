package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestHTTPInlineErrorRootsRetainActualSharedTypes(t *testing.T) {
	for _, tc := range []struct{ schema, name, want string }{
		{`{"allOf":[{"properties":{"message":{"type":"string"}},"required":["message"]},{"properties":{"code":{"type":"integer","default":7}}}]}`, "FetchError400Dto", "val message: kotlin.String,"},
		{`{"oneOf":[{"properties":{"kind":{"const":"a"},"message":{"type":"string"}},"required":["kind","message"]},{"properties":{"kind":{"const":"b"},"code":{"type":"integer"}},"required":["kind","code"]}]}`, "FetchError400Dto", "sealed"},
		{`{"enum":["missing","busy"]}`, "FetchError400Dto", "Unrecognized"},
		{`{"type":"array","items":{"properties":{"message":{"type":"string"}},"required":["message"]}}`, "FetchError400DtoInner", "val message: kotlin.String,"},
		{`{"type":"object","additionalProperties":{"properties":{"message":{"type":"string"}},"required":["message"]}}`, "FetchError400DtoValue", "val message: kotlin.String,"},
	} {
		input := httpDoc(`{"operationId":"fetch","responses":{"204":{},"400":{"content":{"application/json":{"schema":` + tc.schema + `}}}}}`)
		rq := request(input, "Dto")
		first, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, first, tc.name), tc.want) || !strings.Contains(source(t, first, "DefaultApi"), "org.example.models.api.models."+tc.name) {
			t.Fatal("lost inline root type", tc.name)
		}
		second, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("nondeterminism", err)
		}
	}
}

func TestHTTPInlineErrorRootConstructorLimit(t *testing.T) {
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"p%d":{"type":"string"}`, i)
		}
		input := httpDoc(`{"operationId":"fetch","responses":{"204":{},"400":{"content":{"application/json":{"schema":{"allOf":[{"properties":{` + strings.Join(fields, ",") + `}}]}}}}}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if width == 240 {
			if err != nil || !strings.Contains(source(t, got, "FetchError400Dto"), "val p239:") {
				t.Fatal("lost accepted error-root constructor width", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("lost constructor cap", err)
		}
	}
}
