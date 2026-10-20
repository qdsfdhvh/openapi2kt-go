package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRawResponseMediaTypesAndPriorities(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/raw-response-media.json")
	if err != nil {
		t.Fatal(err)
	}
	got := report(t, input, "Dto")
	api := source(t, got, "DefaultApi")
	for _, expected := range []string{
		").body<kotlin.String>()", ").body<kotlin.ByteArray>()", ").body<kotlin.Int>()",
	} {
		if !strings.Contains(api, expected) && !strings.Contains(api, strings.Replace(expected, ").body", "}.body", 1)) {
			t.Fatal("missing raw response type", expected, api)
		}
	}
	if !reflect.DeepEqual(got, report(t, input, "Dto")) {
		t.Fatal("nondeterministic media selection")
	}
}

func TestRawResponseMediaMismatchAndUnselectedCompatibility(t *testing.T) {
	for _, tc := range []struct{ media, schema string }{
		{"text/plain", `{"type":"integer"}`},
		{"text/plain", `{"type":"object","properties":{"name":{"type":"string"}}}`},
		{"application/octet-stream", `{"type":"string"}`},
		{"application/octet-stream", `{"type":"array","items":{"type":"integer"}}`},
	} {
		input := httpDoc(`{"responses":{"200":{"content":{"` + tc.media + `":{"schema":` + tc.schema + `}}}}}`)
		pointer := "/paths/~1ping/get/responses/200/content/" + strings.ReplaceAll(tc.media, "/", "~1") + "/schema"
		failure(t, input, "E_OPERATIONS_UNSUPPORTED", pointer)
	}
	// Existing JSON-preferred documents keep validating other media using their
	// old schema profile; the new raw representation restriction is selected-only.
	input := httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"integer"}},"text/plain":{"schema":{"type":"object","properties":{"v":{"type":"string"}}}}}}}}`)
	api := source(t, report(t, input, ""), "DefaultApi")
	if !strings.Contains(api, "body<kotlin.Int>()") {
		t.Fatal(api)
	}
}

func TestRawResponseMediaExactSpellingAndDroppedSourceValidation(t *testing.T) {
	input := httpDoc(`{"responses":{"200":{"content":{"TEXT/PLAIN":{"schema":{"type":"integer"}},"text/plain":{"schema":{"type":"string"}}}}}}`)
	if api := source(t, report(t, input, ""), "DefaultApi"); !strings.Contains(api, "body<kotlin.String>()") {
		t.Fatal(api)
	}
	bad := httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"integer"}},"text/plain":{"schema":{"type":"string","format":"unsupported"}}}}}}`)
	failure(t, bad, "E_SCHEMA_UNSUPPORTED", "/paths/~1ping/get/responses/200/content/text~1plain/schema/format")
	// Requests and error selection keep their existing media policy.
	failure(t, httpDoc(`{"requestBody":{"content":{"text/plain":{"schema":{"type":"string"}}}},"responses":{"204":{}}}`), "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/requestBody/content")
}
