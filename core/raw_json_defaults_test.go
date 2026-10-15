package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRawJSONFieldDefaults(t *testing.T) {
	// An open dictionary default is a JsonObject; unconstrained and empty-object
	// fields expose JsonElement and accept any JSON value.
	for name, tc := range map[string]struct {
		field, marker, want string
	}{
		"open dictionary": {`{"type":"object","additionalProperties":true,"default":{"a":1,"b":[true,null]}}`, "`openapi-kt object default`(", `{\"a\":1,\"b\":[true,null]}`},
		"empty object":    {`{"type":"object","default":{"x":"y"}}`, "`openapi-kt json default`(", `{\"x\":\"y\"}`},
		"unconstrained":   {`{"default":[1,2,3]}`, "`openapi-kt json default`(", `[1,2,3]`},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{"v":`+tc.field+`}}}`), "Dto"))
			if err != nil {
				t.Fatal(err)
			}
			text := source(t, got, "HolderDto")
			if !strings.Contains(text, tc.marker) || !strings.Contains(text, tc.want) {
				t.Fatal("raw JSON default", text)
			}
			if tc.marker == "`openapi-kt object default`(" && !strings.Contains(text, "private fun `openapi-kt object default`(") {
				t.Fatal("missing object helper", text)
			}
			if tc.marker == "`openapi-kt json default`(" && !strings.Contains(text, "private fun `openapi-kt json default`(") {
				t.Fatal("missing json helper", text)
			}
		})
	}
	// A non-object value for an open dictionary keeps the original Pointer.
	bad := schemas(`{"Holder":{"properties":{"v":{"type":"object","additionalProperties":true,"default":[1]}}}}`)
	r := request(bad, "Dto")
	r.Options.Validation = true
	report, err := core.GenerateWithDiagnostics(r)
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/v/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatalf("raw object wrong kind: %v", err)
	}
}

func TestRawJSONRequestUnionFieldDefaults(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","paths":{"/p":{"post":{"operationId":"p","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","properties":{"u":{"oneOf":[{"type":"string"},{"type":"integer","format":"int32"}],"default":"x"}}}}}},"responses":{"204":{"description":"ok"}}}}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := ""
	for _, file := range got.Artifacts.Files {
		if strings.Contains(file.Content, "`openapi-kt json default`(") {
			text = file.Content
		}
	}
	if !strings.Contains(text, "JsonElement? =") {
		t.Fatal("request-context union raw default", text)
	}
}
