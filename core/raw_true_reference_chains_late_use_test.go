package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestRawTrueReferenceChainsFirstHTTPUnionAndOriginalWarningOwnership(t *testing.T) {
	for _, terminal := range []string{`{"type":"object","nullable":true}`, `{"allOf":[{"type":"object"},{"properties":{}}]}`, `true`} {
		base := `"Value":` + terminal + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":false},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":true}}}`
		for _, paths := range []string{
			`{"/read":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}}}}}}}}`,
			`{"/write":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Hops/properties/value"}}}},"responses":{"204":{}}}}}`,
			`{"/query":{"get":{"parameters":[{"name":"value","in":"query","schema":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}}],"responses":{"204":{}}}}}`,
			`{"/watch":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"oneOf":[{"properties":{"event":{"const":"value"},"data":{"$ref":"#/components/schemas/Hops/properties/value"}},"required":["data"]}]}}}}}}}}`,
		} {
			input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": json.RawMessage(`{` + base + `}`)}, "paths": json.RawMessage(paths)})
			if err != nil {
				t.Fatal(err)
			}
			for _, checked := range []bool{false, true} {
				r := request(input, "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil || strings.Contains(source(t, got, "DefaultApi"), "models.ValueDto") {
					t.Fatal("late raw/true source lost canonical JSON", terminal, checked, err)
				}
				count := 0
				for _, warning := range got.Warnings {
					if warning.Code == "W_JSON_ELEMENT_APPROXIMATION" || warning.Code == "W_BOOLEAN_SCHEMA_PROJECTION" {
						count++
						if warning.Pointer == nil || *warning.Pointer != "/components/schemas/Value" {
							t.Fatal(warning)
						}
					}
				}
				if count != 1 {
					t.Fatal("lateuse duplicated original root warning", terminal, got.Warnings)
				}
			}
		}
		input := schemas(`{` + base + `,"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"value":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}},"required":["value"]},{"properties":{"kind":{"const":"b"},"id":{"type":"integer"}},"required":["id"]}]}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err != nil || !strings.Contains(source(t, got, "ChoiceDto"), "val value: kotlinx.serialization.json.JsonElement") {
			t.Fatal("first union metadata lost JSON source", terminal, err)
		}
	}
	// Preserve the original cover input as an explicit separate cover-chain
	// graduation. Its graph/SCC work and warning are not raw/true permission.
	input, err := os.ReadFile("../tests/fixtures/http/recursive-json.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err = json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	catalog["Hops"] = map[string]any{"properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/ZAny"}}}
	catalog["Bridge"] = map[string]any{"$ref": "#/components/schemas/Hops/properties/value"}
	input, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	got, e := core.GenerateWithDiagnostics(request(input, "Dto"))
	if e != nil || !strings.Contains(source(t, got, "BridgeDto"), "= kotlinx.serialization.json.JsonElement\n") {
		t.Fatal("original cover chain graduation lost", e)
	}
	for _, w := range got.Warnings {
		if w.Code == "W_JSON_ELEMENT_APPROXIMATION" || w.Code == "W_BOOLEAN_SCHEMA_PROJECTION" {
			t.Fatal("cover gained raw/true source role", w)
		}
	}
}
