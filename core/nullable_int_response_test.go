package core_test

import (
	"strings"
	"testing"
)

func TestNullableIntResponseJSONWithoutSSE(t *testing.T) {
	got := report(t, []byte(`{"openapi":"3.1.0","paths":{"/nullable":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"type":"integer","format":"int32","nullable":true}}}}}}},"/plain":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"type":"integer","format":"int32"}}}}}}}}}`), "Dto")
	api := source(t, got, "DefaultApi")
	if !strings.Contains(api, "json.decodeFromString<kotlin.Int?>") || !strings.Contains(api, "bodyAsText()") || !strings.Contains(api, ".body<kotlin.Int>()") {
		t.Fatal(api)
	}
	if !strings.Contains(source(t, got, "ApiClient"), "httpClient, basePath, json") {
		t.Fatal("nullable Int group lost configurable JSON codec")
	}
}
