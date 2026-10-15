package core_test

import (
	"encoding/json"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBuiltinReferenceChainsFirstHTTPAndUnionUse(t *testing.T) {
	// Hops itself contains an ordinary direct named reference. The new chain is
	// first activated by the operation/union, after declarations are recorded.
	base := `"Value":{"type":["string","null"]},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}}`
	for _, path := range []string{
		`{"/read":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}}}}}}}}`,
		`{"/write":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Hops/properties/value"}}}},"responses":{"204":{}}}}}`,
		`{"/query":{"get":{"parameters":[{"name":"value","in":"query","schema":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}}],"responses":{"204":{}}}}}`,
		`{"/watch":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"oneOf":[{"properties":{"event":{"type":"string","const":"value"},"data":{"$ref":"#/components/schemas/Hops/properties/value"}},"required":["data"]}]}}}}}}}}`,
	} {
		input, marshalErr := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": json.RawMessage(`{` + base + `}`)}, "paths": json.RawMessage(path)})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || strings.Contains(source(t, got, "DefaultApi"), "models.ValueDto") {
				t.Fatal("late builtin did not retain backing", checked, err)
			}
		}
	}
	input := schemas(`{` + base + `,"Choice":{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"value":{"$ref":"#/components/schemas/Hops/properties/value"}},"required":["value"]},{"properties":{"kind":{"type":"string","const":"b"},"id":{"type":"integer"}},"required":["id"]}]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || !strings.Contains(source(t, got, "ChoiceDto"), "value: kotlin.String?") {
		t.Fatal("first union tag probe used shape borrowing", err)
	}
}
