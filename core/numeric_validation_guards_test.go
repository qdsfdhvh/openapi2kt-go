package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNumericValidationGuards(t *testing.T) {
	for _, tc := range []struct{ shape, key string }{
		{`{"type":"integer","exclusiveMinimum":true}`, "exclusiveMinimum"},
		{`{"type":"integer","exclusiveMaximum":"x"}`, "exclusiveMaximum"},
		{`{"type":"integer","minimum":2,"exclusiveMaximum":2}`, "exclusiveMaximum"},
		{`{"type":"integer","exclusiveMinimum":2,"exclusiveMaximum":3}`, "exclusiveMaximum"},
		{`{"type":"integer","exclusiveMinimum":2147483647}`, "exclusiveMinimum"},
		{`{"type":"integer","format":"int64","exclusiveMaximum":-9223372036854775808}`, "exclusiveMaximum"},
		{`{"type":"integer","minimum":3,"maximum":3,"multipleOf":2}`, "multipleOf"},
		{`{"type":"integer","minimum":-3,"maximum":-3,"multipleOf":2}`, "multipleOf"},
		{`{"type":"integer","multipleOf":0}`, "multipleOf"},
		{`{"type":"integer","multipleOf":-2}`, "multipleOf"},
		{`{"type":"integer","multipleOf":0.5}`, "multipleOf"},
		{`{"type":"integer","multipleOf":true}`, "multipleOf"},
		{`{"type":"number","format":"float","multipleOf":1e-100}`, "multipleOf"},
		{`{"type":"number","multipleOf":1e-512}`, "multipleOf"},
		{`{"type":"number","exclusiveMinimum":1e-999999}`, "exclusiveMinimum"},
	} {
		aliasFailure(t, fieldDoc(tc.shape), "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/"+tc.key)
	}
	for _, tc := range []struct{ shape, key string }{
		{`{"type":"string","multipleOf":2}`, "multipleOf"},
		{`{"type":"boolean","exclusiveMinimum":0}`, "exclusiveMinimum"},
	} {
		aliasFailure(t, fieldDoc(tc.shape), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/"+tc.key)
	}
	graduated := request(fieldDoc(`{"type":"integer","default":2,"multipleOf":2}`), "")
	graduated.Options.Validation = true
	accepted, e := core.GenerateWithDiagnostics(graduated)
	if e != nil || !strings.Contains(source(t, accepted, "Record"), "value % 2 == 0") || strings.Contains(source(t, accepted, "Record"), "val value: kotlin.Int =") {
		t.Fatal("original default/constraint lost required presence", accepted, e)
	}
	// Each3.1 condition survives independent mapping; don't erase mapped ties.
	req := request(fieldDoc(`{"type":"number","format":"float","minimum":0.300000004,"exclusiveMinimum":0.300000001,"maximum":1,"exclusiveMaximum":1,"multipleOf":0.25}`), "")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, r, "Record")
	for _, keyword := range []string{"below minimum", "exclusiveMinimum", "above maximum", "exclusiveMaximum", "multipleOf"} {
		if !strings.Contains(text, keyword) {
			t.Fatal("dropped independent constraint", keyword, text)
		}
	}
	// New constraints cannot be silently lost to raw projection/discarded slots.
	aliasFailure(t, httpDoc(`{"requestBody":{"content":{"application/json":{"schema":{"anyOf":[{"type":"integer","multipleOf":2},{"type":"boolean"}]}}}},"responses":{"204":{}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1ping/get/requestBody/content/application~1json/schema/anyOf/0/multipleOf")
}

func TestNumericValidationLegacyAndDefaults(t *testing.T) {
	legacy := func(shape string) []byte {
		return []byte(strings.Replace(string(fieldDoc(shape)), `3.1.0`, `3.0.3`, 1))
	}
	for _, tc := range []struct{ shape, key string }{
		{`{"type":"integer","exclusiveMinimum":2}`, "exclusiveMinimum"},
		{`{"type":"integer","exclusiveMaximum":true}`, "exclusiveMaximum"},
		{`{"type":"integer","minimum":2,"maximum":2,"exclusiveMinimum":true}`, "maximum"},
		{`{"type":"integer","minimum":2,"maximum":3,"exclusiveMinimum":true,"exclusiveMaximum":true}`, "exclusiveMaximum"},
	} {
		aliasFailure(t, legacy(tc.shape), "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/"+tc.key)
	}
	req := request(legacy(`{"type":"integer","minimum":2,"maximum":8,"exclusiveMinimum":true,"exclusiveMaximum":true,"multipleOf":2}`), "")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, r, "Record")
	if !strings.Contains(text, "value > 2") || !strings.Contains(text, "value < 8") {
		t.Fatal(text)
	}
	// False modifiers are validated no-ops, not finite-value validators.
	before := report(t, legacy(`{"type":"number"}`), "")
	req = request(legacy(`{"type":"number","exclusiveMinimum":false,"exclusiveMaximum":false}`), "")
	req.Options.Validation = true
	after, err := core.GenerateWithDiagnostics(req)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("false modifiers changed output", err)
	}
	for _, shape := range []string{`{"type":"integer","multipleOf":2}`, `{"type":"integer","exclusiveMinimum":0}`} {
		req := request(fieldDoc(shape), "")
		if _, err := core.Generate(req); err == nil {
			t.Fatal("defaultfalse accepted constraints")
		}
	}
	aliasFailure(t, schemas(`{"Even":{"type":"integer","multipleOf":2},"Record":{"type":"object","properties":{"v":{"type":"string"}},"additionalProperties":{"$ref":"#/components/schemas/Even"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Even/multipleOf")
}
