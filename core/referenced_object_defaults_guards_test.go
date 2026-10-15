package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedObjectDefaultsGuards(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   string
		code    string
		pointer string
	}{
		{
			"target own default",
			`{"Inner":{"type":"object","properties":{"a":{"type":"string"}},"default":{"a":"t"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"a":"x"}}}}}`,
			"E_SCHEMA_UNSUPPORTED", "/components/schemas/Inner/default",
		},
		{
			"unknown key",
			`{"Inner":{"type":"object","properties":{"a":{"type":"string"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"zz":1}}}}}`,
			"E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default/zz",
		},
		{
			"missing required",
			`{"Inner":{"type":"object","required":["a"],"properties":{"a":{"type":"string"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{}}}}}`,
			"E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default",
		},
		{
			"narrowed false null",
			`{"Inner":{"type":["object","null"],"properties":{"a":{"type":"string"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","nullable":false,"default":null}}}}`,
			"E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default",
		},
		{
			"wrong kind",
			`{"Inner":{"type":"object","properties":{"a":{"type":"string"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":[]}}}}`,
			"E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default",
		},
	} {
		r := request(schemas(tc.input), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatalf("%s: want %s at %s, got %v", tc.name, tc.code, tc.pointer, err)
		}
	}
	// Excluded target shapes stay rejected without partial output.
	for _, input := range []string{
		`{"Inner":{"allOf":[{"type":"object","properties":{"a":{"type":"string"}}}],"properties":{"b":{"type":"string"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"b":"x"}}}}}`,
		`{"Inner":{"type":"object","properties":{"tag":{"const":"a"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"tag":"a"}}}}}`,
		`{"Inner":{"type":"object","properties":{"e":{"type":"object","enum":["a"]}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"e":"a"}}}}}`,
		`{"Inner":{"type":"object","properties":{"x":{"type":"string","x-native-type":"bigint"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"x":"1"}}}}}`,
		`{"Inner":{"type":"object"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{}}}}}`,
	} {
		r := request(schemas(input), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatalf("excluded referenced object default leaked: %s -> %v", input, err)
		}
	}
	// Retained validation bounds are enforced with --validation.
	bounded := schemas(`{"Inner":{"type":"object","required":["a"],"properties":{"a":{"type":"string","minLength":3}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"a":"xy"}}}}}`)
	r := request(bounded, "Dto")
	r.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(r); err == nil {
		t.Fatal("short bounded referenced property accepted")
	}
}

func TestReferencedObjectDefaultsBudgets(t *testing.T) {
	big := strings.Repeat("a", 33000)
	large := schemas(`{"Inner":{"type":"object","properties":{"s":{"type":"string"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"s":"` + big + `"}}}}}`)
	if _, err := core.GenerateWithDiagnostics(request(large, "Dto")); err == nil {
		t.Fatal("oversized referenced object default accepted")
	}
}
