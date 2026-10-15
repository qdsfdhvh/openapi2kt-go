package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestRootBorrowedCollectionFirstHTTPUnionAndMethods(t *testing.T) {
	for _, terminal := range []string{`{"type":"array","nullable":true,"items":{"type":["string","null"]}}`, `{"additionalProperties":{"type":["string","null"]},"nullable":true}`} {
		base := `"Source":{"properties":{"leaf":` + terminal + `}},"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Bridge","nullable":false}}}`
		for _, paths := range []string{
			`{"/read":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Bridge","nullable":false}}}}}}}}`,
			`{"/write":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Hops/properties/value"}}}},"responses":{"204":{}}}}}`,
			`{"/query":{"get":{"parameters":[{"name":"value","in":"query","schema":{"$ref":"#/components/schemas/Bridge","nullable":false}}],"responses":{"204":{}}}}}`,
			`{"/watch":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"oneOf":[{"properties":{"event":{"const":"value"},"data":{"$ref":"#/components/schemas/Bridge"}},"required":["data"]}]}}}}}}}}`,
		} {
			input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": json.RawMessage(`{` + base + `}`)}, "paths": json.RawMessage(paths)})
			if err != nil {
				t.Fatal(err)
			}
			for _, checked := range []bool{false, true} {
				r := request(input, "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil || strings.Contains(source(t, got, "DefaultApi"), "models.BridgeDto>") {
					t.Fatal("late borrowedbacking lost", terminal, checked, err)
				}
			}
		}
		input := schemas(`{` + base + `,"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"value":{"$ref":"#/components/schemas/Bridge","nullable":false}},"required":["value"]},{"properties":{"kind":{"const":"b"},"id":{"type":"string"}},"required":["id"]}]}}`)
		if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
			t.Fatal("union original borrowedroot metadata lost", err)
		}
	}
	for _, stream := range []bool{false, true} {
		limit := 240
		response := `{"204":{}}`
		if stream {
			limit = 239
			response = `{"200":{"content":{"text/event-stream":{"schema":{"properties":{}}}}}}`
		}
		for _, count := range []int{limit, limit + 1} {
			var params []string
			for i := 0; i < count; i++ {
				params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Bridge","nullable":false}}`, i))
			}
			input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"list":{"type":"array","items":{"type":"string","minLength":2},"minItems":1}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/list"}}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":` + response + `}}}}`)
			r := request(input, "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			if count == limit {
				if err != nil {
					t.Fatal(stream, count, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_OPERATION_LIMIT" || len(got.Artifacts.Files) != 0 {
				t.Fatal("borrowed method slots changed", stream, count, err)
			}
		}
	}
}
