package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestHTTPInlineErrorsValidateEverySourceAndExcludedGroups(t *testing.T) {
	input := httpDoc(`{"operationId":"fetch","responses":{"204":{},"400":{"content":{"application/json":{"schema":{"properties":{"message":{"type":"string"}},"required":["message"]}},"application/problem+json":{"schema":{"properties":{"id":{"type":"string","format":"unsupported"}}}}}}}}`)
	for _, excluded := range []bool{false, true} {
		rq := request(input, "Dto")
		if excluded {
			rq.Options.ExcludeAPIs = []string{"DefaultApi"}
		}
		got, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(got.Artifacts.Files) != 0 || *err.(core.Diagnostics)[0].Pointer != "/paths/~1ping/get/responses/400/content/application~1problem+json/schema/properties/id/format" {
			t.Fatal("discarded error source lost", got, err)
		}
	}
}

func TestHTTPInlineErrorsNullableBorrowedAndOptimization(t *testing.T) {
	shape := `{"type":["object","null"],"properties":{"message":{"type":"string","default":"missing"},"nested":{"properties":{"id":{"type":"string"}}}},"required":["message"]}`
	op := func(id string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"204":{},"400":{"$ref":"#/components/responses/Problem"}}}}`
	}
	doc := []byte(`{"openapi":"3.1.0","components":{"responses":{"Problem":{"content":{"application/json":{"schema":` + shape + `}}}}},"paths":{"/a":` + op("alpha") + `,"/b":` + op("beta") + `}}`)
	rq := request(doc, "Dto")
	rq.Options.HoistSharedShapes = true
	rq.Options.DeduplicateModels = true
	rq.Options.MergeIdenticalModels = true
	rq.Options.LockHoistedNames = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"AlphaError400Dto", "BetaError400Dto"} {
		if !strings.Contains(source(t, got, name), "typealias "+name+" = org.example.models.models.Error400Dto") {
			t.Fatal("error model did not relocate", name, got)
		}
	}
	if !strings.Contains(source(t, got, "DefaultApi"), "org.example.models.models.Error400Dto?") || !strings.Contains(source(t, got, "Error400Dto"), "val message: kotlin.String,") {
		t.Fatal("relocated error type nullable/presence lost", got)
	}
}
