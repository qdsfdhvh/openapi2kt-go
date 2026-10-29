package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func tupleRequest(t *testing.T, schema any) core.GenerationRequest {
	t.Helper()
	input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Probe": schema}}})
	if err != nil {
		t.Fatal(err)
	}
	return request(input, "")
}

func TestPrimitiveTupleExactSignatures(t *testing.T) {
	for _, test := range []struct {
		prefix, items string
		want          string
	}{
		{`[{"type":"string"},{"type":"string"}]`, "", "kotlin.String"},
		{`[{"type":"integer","format":"int32"},{"type":"integer"}]`, "", "kotlinx.serialization.json.JsonElement"},
		{`[{"type":"string","format":""},{"type":"string"}]`, "", "kotlinx.serialization.json.JsonElement"},
		{`[{"type":"string","nullable":true},{"type":["null","string"]}]`, "", "kotlin.String?"},
		{`[{"type":"string","nullable":false},{"type":"string","nullable":true}]`, "", "kotlinx.serialization.json.JsonElement"},
		{`[{"type":"integer"},{"type":"boolean"}]`, `,"items":{"type":"number","format":"float"}`, "kotlin.Float"},
		// Preserve the exact former date-member rejection as a positive.
		{`[{"type":"string","format":"date"}]`, "", "kotlinx.datetime.LocalDate"},
		{`[{"type":"string","format":"date-time"},{"type":"string","format":"date-time"}]`, "", "kotlin.time.Instant"},
		{`[{"type":"string","format":"date","nullable":true},{"type":["null","string"],"format":"date"}]`, "", "kotlinx.datetime.LocalDate?"},
		{`[{"type":"string","format":"date"},{"type":"string","format":"date-time"}]`, "", "kotlinx.serialization.json.JsonElement"},
		{`[{"type":"string","format":"date"},{"type":"string"}]`, `,"items":{"type":"string","format":"date-time","nullable":true}`, "kotlin.time.Instant?"},
		{`[{"type":"string"}]`, `,"items":false`, "kotlin.String"},
		{`[{"type":"string"}]`, `,"items":true`, "kotlinx.serialization.json.JsonElement"},
	} {
		var schema any
		if err := json.Unmarshal([]byte(`{"type":"array","prefixItems":`+test.prefix+test.items+`}`), &schema); err != nil {
			t.Fatal(err)
		}
		r := tupleRequest(t, schema)
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, got, "models/Probe"), "List<"+test.want+">") {
			t.Fatal(test, got)
		}
		if len(got.Warnings) != 1 || got.Warnings[0].Code != "W_TUPLE_APPROXIMATION" || got.Warnings[0].Pointer == nil || *got.Warnings[0].Pointer != "/components/schemas/Probe/prefixItems" {
			t.Fatal(got.Warnings)
		}
	}
}

func TestPrimitiveTupleNoDropGuards(t *testing.T) {
	for _, test := range []struct{ schema, code, pointer string }{
		{`{"type":"array","prefixItems":[]}`, "E_SCHEMA_INVALID", "/prefixItems"},
		{`{"type":"array","prefixItems":{}}`, "E_SCHEMA_INVALID", "/prefixItems"},
		{`{"type":"array","prefixItems":[{"type":"string","format":3}]}`, "E_SCHEMA_INVALID", "/prefixItems/0"},
		{`{"type":"array","prefixItems":[{"type":"string"},{"type":"integer","minimum":1}],"items":{"type":"string"}}`, "E_SCHEMA_UNSUPPORTED", "/prefixItems/1/minimum"},
		{`{"type":"array","prefixItems":[{"type":"string","unknownWire":true}],"items":{"type":"string"}}`, "E_SCHEMA_UNSUPPORTED", "/prefixItems/0/unknownWire"},
		{`{"type":"array","prefixItems":[{"type":"string","x-native-type":"org.example.Provider"}]}`, "E_SCHEMA_UNSUPPORTED", "/prefixItems/0/x-native-type"},
		{`{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Probe"}]}`, "E_REF_CYCLE", ""},
		{`{"type":"array","prefixItems":[{"type":"string","format":"date","minLength":1}],"items":{"type":"string"}}`, "E_SCHEMA_UNSUPPORTED", "/prefixItems/0/minLength"},
		{`{"type":"array","prefixItems":[{"type":"string","format":"date-time","default":"2026-01-01T00:00:00Z"}],"items":{"type":"string"}}`, "E_SCHEMA_UNSUPPORTED", "/prefixItems/0/default"},
		{`{"type":"array","prefixItems":[{"type":"string","format":"binary"}]}`, "E_SCHEMA_UNSUPPORTED", "/prefixItems/0"},
		{`{"type":"array","prefixItems":[{"type":"string","format":"date"}],"uniqueItems":true}`, "E_SCHEMA_UNSUPPORTED", "/uniqueItems"},
	} {
		for _, checked := range []bool{false, true} {
			var schema any
			if err := json.Unmarshal([]byte(test.schema), &schema); err != nil {
				t.Fatal(err)
			}
			r := tupleRequest(t, schema)
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("partial or accepted", test, got, err)
			}
			d, ok := err.(core.Diagnostics)
			if !ok || len(d) != 1 || d[0].Code != test.code || d[0].Pointer == nil || *d[0].Pointer != "/components/schemas/Probe"+test.pointer {
				t.Fatal(test, err)
			}
		}
	}
	r := tupleRequest(t, map[string]any{"type": "array", "prefixItems": []any{map[string]any{"type": "string"}}})
	r.Document = []byte(strings.Replace(string(r.Document), "3.1.0", "3.0.3", 1))
	if _, err := core.Generate(r); err == nil {
		t.Fatal("3.0 prefix broadened")
	}
}

func TestPrimitiveTupleDepthAndConstructorBudgets(t *testing.T) {
	for _, depth := range []int{48, 49} {
		var schema any = map[string]any{"type": "array", "prefixItems": []any{map[string]any{"type": "string"}, map[string]any{"type": "boolean"}}}
		for i := 1; i < depth; i++ {
			schema = map[string]any{"type": "array", "items": schema}
		}
		r := tupleRequest(t, schema)
		got, err := core.GenerateWithDiagnostics(r)
		if depth == 48 && err != nil || depth == 49 && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(depth, err)
		}
	}
	for _, width := range []int{240, 241} {
		props := map[string]any{}
		required := []string{}
		for i := 0; i < width; i++ {
			name := strings.Repeat("x", i+1)
			required = append(required, name)
			props[name] = map[string]any{"type": "array", "prefixItems": []any{map[string]any{"type": "integer", "format": "int64"}}}
		}
		r := tupleRequest(t, map[string]any{"type": "object", "properties": props, "required": required})
		got, err := core.GenerateWithDiagnostics(r)
		if width == 240 && err != nil || width == 241 && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(width, err)
		}
	}
}
