package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRawRequestMedia(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/raw-request-media.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.ModelStyle = style
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(err)
			}
			api := source(t, got, "DefaultApi")
			for _, want := range []string{`parse("text/plain")`, `parse("TEXT/PLAIN; charset=ISO-8859-1")`, `parse("application/octet-stream")`, `input: kotlin.ByteArray`, `.Application.Json`, "FormDataContent"} {
				if !strings.Contains(api, want) {
					t.Fatal("raw request/priority missing", want, api)
				}
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("raw request nondeterminism", err)
			}
		}
	}
}

func TestRawRequestMediaSelectedOnlyBoundary(t *testing.T) {
	for _, tc := range []struct{ media, schema string }{{"text/plain", `{"type":"integer"}`}, {"text/plain", `{"type":"object","properties":{"value":{"type":"string"}}}`}, {"application/octet-stream", `{"type":"string"}`}, {"application/octet-stream", `{"type":"array","items":{"type":"integer"}}`}, {"text/plain", `{"type":["string","null"]}`}} {
		input := httpDoc(`{"requestBody":{"required":true,"content":{"` + tc.media + `":{"schema":` + tc.schema + `}}},"responses":{"204":{}}}`)
		failure(t, input, "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/requestBody/content/"+strings.ReplaceAll(tc.media, "/", "~1")+"/schema")
	}
	// Optional non-null raw bodies graduate through TestOptionalRawRequestPresence.
	bad := httpDoc(`{"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"integer"}},"text/plain":{"schema":{"type":"string","format":"unsupported"}}}},"responses":{"204":{}}}`)
	failure(t, bad, "E_SCHEMA_UNSUPPORTED", "/paths/~1ping/get/requestBody/content/text~1plain/schema/format")
	// Exact spelling wins; the discarded case variant keeps original shared
	// schema validation and does not acquire the selected raw type restriction.
	input := httpDoc(`{"requestBody":{"required":true,"content":{"TEXT/PLAIN":{"schema":{"type":"integer"}},"text/plain":{"schema":{"type":"string"}}}},"responses":{"204":{}}}`)
	if _, err := core.Generate(request(input, "")); err != nil {
		t.Fatal(err)
	}
	malformed := httpDoc(`{"requestBody":{"required":true,"content":{"text/plain; broken":{"schema":{"type":"string"}}}},"responses":{"204":{}}}`)
	failure(t, malformed, "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/requestBody/content/text~1plain; broken")
}
