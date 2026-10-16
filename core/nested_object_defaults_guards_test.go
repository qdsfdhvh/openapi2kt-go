package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedObjectDefaultsGuards(t *testing.T) {
	for _, tc := range []struct {
		field string
		code  string
		tail  string
	}{
		{`{"type":"object","properties":{"a":{"type":"string"}},"required":["a"],"default":{}}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"object","properties":{"a":{"type":"string"}},"default":{"b":1}}`, "E_SCHEMA_INVALID", "/default/b"},
		{`{"type":"object","properties":{"a":{"type":"string"}},"default":{"a":null}}`, "E_SCHEMA_INVALID", "/default/a"},
		{`{"type":"object","properties":{"a":{"type":"string"}},"default":"x"}`, "E_SCHEMA_INVALID", "/default"},
		{`{"properties":{"id":{"type":"string"}},"default":[]}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"object","properties":{"a":{"type":"string"}},"default":{"a":1}}`, "E_SCHEMA_INVALID", "/default/a"},
		{`{"type":"object","properties":{"a":{"type":"string"}},"allOf":[{"properties":{"a":{"type":"string"}}}],"default":{"a":"x"}}`, "E_SCHEMA_UNSUPPORTED", "/default"},
		{`{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":{"type":"integer"},"default":{"a":"x"}}`, "E_SCHEMA_UNSUPPORTED", "/default"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.field), "Dto")
			r.Options.Validation = validation
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			want := "/components/schemas/Record/properties/value" + tc.tail
			if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != want || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatalf("want %s at %s (validation=%v), got %v", tc.code, want, validation, err)
			}
		}
	}
	// Retained validation bounds only participate with --validation.
	for _, validation := range []bool{false, true} {
		r := request(fieldDoc(`{"type":"object","properties":{"a":{"type":"string","minLength":3}},"default":{"a":"xy"}}`), "Dto")
		r.Options.Validation = validation
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatalf("short bounded property accepted (validation=%v)", validation)
		}
	}
	// Excluded sub-roles stay rejected without partial output.
	for _, input := range []string{
		// A container child default is not an outer object default.
		`{"Holder":{"properties":{"value":{"type":"object","properties":{"rows":{"type":"array","items":{"type":"object","properties":{"v":{"type":"integer"}},"default":{"v":1}}}}},"default":{"rows":[]}}}}}`,
	} {
		r := request(schemas(input), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatalf("excluded object default role leaked: %s -> %v", input, err)
		}
	}
	// Discarded allOf defaults still validate before a later override wins.
	valid := schemas(`{"Holder":{"allOf":[{"properties":{"value":{"type":"object","properties":{"a":{"type":"string"}},"default":{"a":"x"}}}},{"properties":{"value":{"type":"string"}}}]}}`)
	if _, err := core.GenerateWithDiagnostics(request(valid, "Dto")); err != nil {
		t.Fatal("valid discarded object default rejected", err)
	}
	invalid := schemas(`{"Holder":{"allOf":[{"properties":{"value":{"type":"object","properties":{"a":{"type":"string"}},"default":{"b":"x"}}}},{"properties":{"value":{"type":"string"}}}]}}`)
	if _, err := core.GenerateWithDiagnostics(request(invalid, "Dto")); err == nil {
		t.Fatal("invalid discarded object default accepted")
	}
}

func TestNestedObjectDefaultsBudgets(t *testing.T) {
	// 32 KiB canonical encoding limit.
	big := strings.Repeat("a", 33000)
	large := schemas(`{"Holder":{"properties":{"value":{"type":"object","properties":{"s":{"type":"string"}},"default":{"s":"` + big + `"}}}}}`)
	if _, err := core.GenerateWithDiagnostics(request(large, "Dto")); err == nil {
		t.Fatal("oversized object default accepted")
	}
	// Schema depth stays bounded for nested plain object trees.
	schema := `{"type":"string"}`
	payload := `"x"`
	for i := 0; i < 50; i++ {
		schema = `{"type":"object","properties":{"n":` + schema + `}}`
		payload = `{"n":` + payload + `}`
	}
	deep := schemas(`{"Holder":{"properties":{"value":` + schema + `,"default":` + payload + `}}}`)
	if _, err := core.GenerateWithDiagnostics(request(deep, "Dto")); err == nil {
		t.Fatal("over-depth object default accepted")
	}
}
