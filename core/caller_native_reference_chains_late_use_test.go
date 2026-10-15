package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestCallerNativeReferenceChainsFirstHTTPUnionAndPathLocalIntent(t *testing.T) {
	base := `"Value":{"x-native-type":"org.example.scalarvalues.Money","nullable":true},"Alias":{"$ref":"#/components/schemas/Value","nullable":false},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":true}}}`
	for _, path := range []string{
		`{"/read":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}}}}}}}}`,
		`{"/write":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Hops/properties/value"}}}},"responses":{"204":{}}}}}`,
		`{"/query":{"get":{"parameters":[{"name":"value","in":"query","schema":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}}],"responses":{"204":{}}}}}`,
		`{"/watch":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"oneOf":[{"properties":{"event":{"const":"value"},"data":{"$ref":"#/components/schemas/Hops/properties/value"}},"required":["data"]}]}}}}}}}}`,
	} {
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": json.RawMessage(`{` + base + `}`)}, "paths": json.RawMessage(path)})
		if err != nil {
			t.Fatal(err)
		}
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || strings.Contains(source(t, got, "DefaultApi"), "LongAsStringSerializer") || strings.Contains(source(t, got, "DefaultApi"), "models.ValueDto") {
				t.Fatal("late operation lost transparent numeric backing", checked, err)
			}
		}
	}
	input := schemas(`{` + base + `,"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"value":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}},"required":["value"]},{"properties":{"kind":{"const":"b"},"id":{"type":"integer"}},"required":["id"]}]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || !strings.Contains(source(t, got, "ChoiceDto"), "val value: org.example.scalarvalues.Money") {
		t.Fatal("first union field source probe lost callerNative", err)
	}
	for _, hop := range []string{`{"$ref":"#/components/schemas/Bridge"}`, `{"$ref":"#/components/schemas/Hops/properties/value"}`, `{"$ref":"#/components/schemas/Value","default":"x"}`} {
		input := schemas(`{"Value":{"x-native-type":"org.example.scalarvalues.Money"},"Hops":{"properties":{"value":` + hop + `}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`)
		if got, err := core.GenerateWithDiagnostics(request(input, "Dto")); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("cycle/sibling accepted", hop, err)
		}
	}
}
