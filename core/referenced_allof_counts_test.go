package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedNestedAllOfCountsConstrainWholeObject(t *testing.T) {
	for _, catalog := range []string{
		`{"Lower":{"properties":{"value":{"type":"string"}},"minProperties":1},"Record":{"allOf":[{"$ref":"#/components/schemas/Lower"},{"properties":{"other":{"type":"boolean"}},"maxProperties":2}]}}`,
		`{"Lower":{"properties":{"value":{"type":"string"}},"minProperties":1},"Alias":{"$ref":"#/components/schemas/Lower"},"Record":{"allOf":[{"$ref":"#/components/schemas/Alias"},{"properties":{"other":{"type":"boolean"}},"maxProperties":2}]}}`,
		`{"Record":{"allOf":[{"type":"object","allOf":[{"properties":{"value":{"type":"string"}},"minProperties":1}]},{"properties":{"other":{"type":"boolean"}},"maxProperties":2}]}}`,
		`{"Lower":{"properties":{"value":{"type":"string"}}},"Record":{"allOf":[{"$ref":"#/components/schemas/Lower"},{"type":"object","allOf":[{"properties":{"other":{"type":"boolean"}}}]}],"minProperties":1,"maxProperties":2}}`,
	} {
		r := request(schemas(catalog), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e != nil {
			t.Fatal(catalog, e)
		}
		text := source(t, got, "RecordDto")
		for _, marker := range []string{"generatedSerializer()", "element.size < 1", "element.size > 2", "public val value: kotlin.String? = null", "public val other: kotlin.Boolean? = null"} {
			if !strings.Contains(text, marker) {
				t.Fatal(marker, text)
			}
		}
		if strings.Contains(text, "init {") || strings.Count(text, "element.size < 1") != 1 || strings.Count(text, "element.size > 2") != 1 {
			t.Fatal("Count barriers or source-count output changed", text)
		}
		again, e := core.GenerateWithDiagnostics(r)
		if e != nil || !reflect.DeepEqual(got, again) {
			t.Fatal(e)
		}
		r.Options.Validation = false
		if _, e := core.GenerateWithDiagnostics(r); e == nil {
			t.Fatal("Default-off admitted whole-object count")
		}
	}
}

func TestReferencedNestedAllOfCountsRetainPostorderAndLocalValidationPriority(t *testing.T) {
	for _, tc := range []struct{ catalog, pointer string }{
		{`{"Record":{"allOf":[{"properties":{"value":{"type":"string"}},"minProperties":3},{"type":"object","allOf":[{"properties":{"value":{"type":"string"}},"maxProperties":2},{"properties":{"value":{"type":"string"}},"maxProperties":4}]}]}}`, "/components/schemas/Record/allOf/1/allOf/0/maxProperties"},
		{`{"A":{"properties":{"value":{"type":"string"}},"minProperties":3},"B":{"properties":{"value":{"type":"string"}},"maxProperties":2},"Record":{"allOf":[{"$ref":"#/components/schemas/A"},{"type":"object","allOf":[{"$ref":"#/components/schemas/B"}]}]}}`, "/components/schemas/B/maxProperties"},
		{`{"Record":{"allOf":[{"properties":{"value":{"type":"string"}},"minProperties":3},{"properties":{"value":{"type":"string"}},"maxProperties":2},{"type":"object","allOf":[{"properties":{"value":{"type":"string"}},"minProperties":1.0}]}]}}`, "/components/schemas/Record/allOf/2/allOf/0/minProperties"},
	} {
		r := request(schemas(tc.catalog), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *e.(core.Diagnostics)[0].Pointer != tc.pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc, e)
		}
	}
}
