package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedNestedAllOfBoundsKeepOriginalScalarConjunction(t *testing.T) {
	for _, catalog := range []string{
		`{"Lower":{"properties":{"value":{"type":"integer","minimum":2}},"required":["value"]},"Record":{"allOf":[{"$ref":"#/components/schemas/Lower"},{"properties":{"value":{"type":"integer","maximum":4}}}]}}`,
		`{"Lower":{"properties":{"value":{"type":"integer","minimum":2}},"required":["value"]},"Alias":{"$ref":"#/components/schemas/Lower"},"Record":{"allOf":[{"$ref":"#/components/schemas/Alias"},{"properties":{"value":{"type":"integer","maximum":4}}}]}}`,
		`{"Record":{"allOf":[{"allOf":[{"properties":{"value":{"type":"integer","minimum":2}},"required":["value"]}]},{"properties":{"value":{"type":"integer","maximum":4}}}]}}`,
		`{"Lower":{"properties":{"value":{"type":"integer","minimum":2}},"required":["value"]},"Upper":{"properties":{"value":{"type":"integer","maximum":4}}},"Record":{"allOf":[{"$ref":"#/components/schemas/Lower"},{"$ref":"#/components/schemas/Upper"}]}}`,
	} {
		r := request(schemas(catalog), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e != nil {
			t.Fatal(catalog, e)
		}
		text := source(t, got, "RecordDto")
		for _, token := range []string{"public val value: kotlin.Int,", "value >= 2", "value <= 4", "private fun openapiKtValidatePrimitive"} {
			if !strings.Contains(text, token) {
				t.Fatal(token, text)
			}
		}
		again, e := core.GenerateWithDiagnostics(r)
		if e != nil || !reflect.DeepEqual(got, again) {
			t.Fatal(e)
		}
		r.Options.Validation = false
		if _, e := core.GenerateWithDiagnostics(r); e == nil {
			t.Fatal("Default-off admitted bounded source")
		}
	}
}

func TestReferencedNestedAllOfBoundsKeepWinnerOrderAndReferencedClosingSources(t *testing.T) {
	input := schemas(`{"Lower":{"properties":{"first":{"type":"string","minLength":1},"last":{"type":"integer"}},"required":["first"]},"Record":{"allOf":[{"$ref":"#/components/schemas/Lower"},{"type":"object","allOf":[{"properties":{"first":{"type":"string","maxLength":4}}}]}],"properties":{"first":{"type":"string","minLength":2,"description":"winning source"},"own":{"type":"boolean"}}}}`)
	r := request(input, "Dto")
	r.Options.Validation = true
	got, e := core.GenerateWithDiagnostics(r)
	if e != nil {
		t.Fatal(e)
	}
	text := source(t, got, "RecordDto")
	if !strings.Contains(text, "public val first: kotlin.String,") || !strings.Contains(text, "length >= 2") || !strings.Contains(text, "length <= 4") || strings.Index(text, `SerialName("first")`) > strings.Index(text, `SerialName("last")`) || strings.Index(text, `SerialName("last")`) > strings.Index(text, `SerialName("own")`) {
		t.Fatal("Original winning/presence/positional contract changed", text)
	}
	for _, kind := range []string{"string", "integer"} {
		lower, upper := `"minLength":4`, `"maxLength":3`
		if kind == "integer" {
			lower, upper = `"minimum":4`, `"maximum":3`
		}
		catalog := `{"A":{"properties":{"value":{"type":"` + kind + `",` + lower + `}}},"B":{"properties":{"value":{"type":"` + kind + `",` + upper + `}}},"C":{"properties":{"value":{"type":"` + kind + `",` + upper + `}}},"Record":{"allOf":[{"$ref":"#/components/schemas/A"},{"type":"object","allOf":[{"$ref":"#/components/schemas/B"},{"$ref":"#/components/schemas/C"}]}]}}`
		r.Document = schemas(catalog)
		failed, e := core.GenerateWithDiagnostics(r)
		keyword := "maxLength"
		if kind == "integer" {
			keyword = "maximum"
		}
		if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/B/properties/value/"+keyword || len(failed.Artifacts.Files) != 0 {
			t.Fatal("First real referenced closing source not retained", kind, e)
		}
	}
}

func TestReferencedNestedAllOfBoundsKeepOriginalContributionOrder(t *testing.T) {
	input := schemas(`{"Record":{"allOf":[{"properties":{"value":{"type":"string","minLength":4}}},{"allOf":[{"properties":{"value":{"type":"string","maxLength":3}}},{"properties":{"value":{"type":"string","maxLength":6}}}]}]}}`)
	r := request(input, "Dto")
	r.Options.Validation = true
	got, e := core.GenerateWithDiagnostics(r)
	if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/allOf/1/allOf/0/properties/value/maxLength" || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("Original A-B-C closing source not retained", e)
	}
}
