package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestOptionalRawRequestPresence(t *testing.T) {
	for _, bodyRequired := range []string{"", `"required":false,`} {
		for _, tc := range []struct{ media, schema, typ string }{
			{"text/plain", `{"type":"string"}`, "kotlin.String"},
			{"application/octet-stream", `{"type":"string","format":"binary"}`, "kotlin.ByteArray"},
		} {
			input := httpDoc(`{"requestBody":{` + bodyRequired + `"content":{"` + tc.media + `":{"schema":` + tc.schema + `}}},"responses":{"204":{}}}`)
			for _, checked := range []bool{false, true} {
				rq := request(input, "Dto")
				rq.Options.Validation = checked
				report, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("optional raw request rejected", tc.media, err)
				}
				api := source(t, report, "DefaultApi")
				if !strings.Contains(api, "input: "+tc.typ+"? = null") || !strings.Contains(api, "if (input != null) {") {
					t.Fatal("optional raw body lost explicit absence", api)
				}
			}
		}
	}
}

func TestOptionalRawRequestNullableSchemaStillRejected(t *testing.T) {
	for _, required := range []string{"", `"required":false,`, `"required":true,`} {
		for _, tc := range []struct{ media, schema string }{
			{"text/plain", `{"type":["string","null"]}`},
			{"application/octet-stream", `{"type":"string","format":"binary","nullable":true}`},
		} {
			failure(t, httpDoc(`{"requestBody":{`+required+`"content":{"`+tc.media+`":{"schema":`+tc.schema+`}}},"responses":{"204":{}}}`), "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/requestBody/content/"+strings.ReplaceAll(tc.media, "/", "~1")+"/schema")
		}
	}
}
