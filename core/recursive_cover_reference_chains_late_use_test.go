package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestRecursiveCoverReferenceChainsFirstHTTPUnionAndOriginalGraphGrammar(t *testing.T) {
	base := `"Value":` + coverChainRoot("Other", "Other") + `,"Other":` + coverChainRoot("Value", "Value") + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":false},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":true}}}`
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
				t.Fatal("lateusecover nativeJSONlost", checked, err)
			}
			count := 0
			for _, w := range got.Warnings {
				if w.Code == "W_RECURSIVE_JSON_PROJECTION" {
					count++
					if w.Pointer == nil || (*w.Pointer != "/components/schemas/Value" && *w.Pointer != "/components/schemas/Other") {
						t.Fatal(w)
					}
				}
			}
			if count != 2 {
				t.Fatal("late cover warnings", got.Warnings)
			}
		}
	}
	input := schemas(`{` + base + `,"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"value":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}},"required":["value"]},{"properties":{"kind":{"const":"b"},"id":{"type":"string"}},"required":["id"]}]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || !strings.Contains(source(t, got, "ChoiceDto"), "val value: kotlinx.serialization.json.JsonElement") {
		t.Fatal("firstunioncover metadata probe lost", err)
	}
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ replacement, pointer string }{
			{strings.Replace(coverChainRoot("Value", "Value"), `{"type":"string"}`, `{"type":"string","minLength":1}`, 1), "/components/schemas/Other/oneOf/1/minLength"},
			{strings.Replace(coverChainRoot("Value", "Value"), `"#/components/schemas/Value"`, `"#/components/schemas/Value","nullable":false`, 1), "/components/schemas/Other/oneOf/4/items/nullable"},
		} {
			input := schemas(`{"Value":` + coverChainRoot("Other", "Other") + `,"Other":` + tc.replacement + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`)
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if tc.pointer == "/components/schemas/Other/oneOf/4/items/nullable" {
				if err != nil || !strings.Contains(source(t, got, "ValueDto"), "kotlinx.serialization.json.JsonElement") {
					t.Fatal("original descendant nullable edge rejected", err)
				}
				continue
			}
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
				t.Fatal("original descendant grammar lost", checked, tc, err)
			}
		}
		for _, hop := range []string{`{"$ref":"#/components/schemas/Hops/properties/value"}`, `{"$ref":"#/components/schemas/Bridge"}`, `{"$ref":"#/components/schemas/Value","default":null}`} {
			r := request(schemas(`{"Value":`+coverChainRoot("Value", "Value")+`,"Hops":{"properties":{"value":`+hop+`}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("coverchaincycle/siblingaccepted", hop, err)
			}
		}
	}
}
