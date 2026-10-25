package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestOptionalFormRequestPresence(t *testing.T) {
	for _, bodyRequired := range []string{"", `"required":false,`} {
		input := httpDoc(`{"requestBody":{` + bodyRequired + `"content":{"application/x-www-form-urlencoded":{"schema":{"type":"object","properties":{"v":{"type":"string"}},"required":["v"]}}}},"responses":{"204":{}}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("optional form rejected", err)
			}
			api := source(t, got, "DefaultApi")
			if !strings.Contains(api, "InputDto? = null") || !strings.Contains(api, "if (input != null) {") || !strings.Contains(api, "FormDataContent") {
				t.Fatal("optional form lost absence guard", api)
			}
		}
	}
}
