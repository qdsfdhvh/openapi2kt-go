package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestHTTPInlineErrorDTOsRetainOriginalStatusAndBodyType(t *testing.T) {
	input := httpDoc(`{"operationId":"fetch","responses":{"200":{"content":{"application/json":{"schema":{"type":"string"}}}},"400":{"content":{"application/json":{"schema":{"properties":{"message":{"type":"string"}},"required":["message"]}}}},"5XX":{"content":{"application/problem+json":{"schema":{"properties":{"message":{"type":"string"}},"required":["message"]}}}},"default":{"content":{"application/json":{"schema":{"properties":{"message":{"type":"string"}},"required":["message"]}}}}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	api := source(t, got, "DefaultApi")
	for _, name := range []string{"FetchError400Dto", "FetchError5XXDto", "FetchErrorDefaultDto"} {
		text := source(t, got, name)
		if !strings.Contains(text, "val message: kotlin.String,") || !strings.Contains(api, "org.example.models.api.models."+name) {
			t.Fatal("inline error type or status binding lost", name, api, text)
		}
	}
}
