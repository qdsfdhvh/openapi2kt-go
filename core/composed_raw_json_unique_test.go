package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestJSONUniqueComposedRawReferences(t *testing.T) {
	// Exact formerly rejected input, now admitted by retained empty Model authority.
	rq := request(schemas(`{"Value":{"allOf":[{"type":"object"},{"properties":{}}]},"Record":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Value"},"uniqueItems":true}}}}`), "Dto")
	rq.Options.Validation = true
	if _, err := core.Generate(rq); err != nil {
		t.Fatal(err)
	}
	for _, terminal := range []string{
		`{"allOf":[{"type":"object"},{"properties":{}}]}`,
		`{"allOf":[{"allOf":[{"type":"object"},{"properties":{}}]},{"type":"object"}]}`,
		`{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{}}]}`,
	} {
		for _, ref := range []string{"Value", "Alias", "Hops/properties/ref", "Bridge"} {
			doc := schemas(`{"Base":{"type":"object"},"Value":` + terminal + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Hops":{"properties":{"ref":{"$ref":"#/components/schemas/Alias","nullable":false}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/ref"},"Record":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/` + ref + `"},"uniqueItems":true}}}}`)
			rq := request(doc, "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(terminal, ref, err)
			}
			if !strings.Contains(source(t, got, "RecordDto"), "`openapi-kt unique JSON items`(value)") {
				t.Fatal("composed raw uniqueness lost", ref)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("composed source/output drift", err)
			}
			rq.Options.Validation = false
			if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("composed raw uniqueness missing opt-in", ref, err)
			}
		}
	}
}

func TestJSONUniqueComposedRawBudgetsAndNoDrop(t *testing.T) {
	terminal := `{"allOf":[{"type":"object"},{"properties":{}}]}`
	enumUniqueCheckedBudgets(t, terminal)
	enumUniqueSourceCacheBudgets(t, terminal)
	jsonUniqueReferencedNoDropAndNoop(t, `"Value":`+terminal+`, `)
}

func TestJSONUniqueComposedRawExcludedSources(t *testing.T) {
	for _, terminal := range []string{
		`{"allOf":[{"properties":{"id":{"type":"string"}}},{"properties":{}}]}`,
		`{"allOf":[{"properties":{"kind":{"const":"only"}}},{"type":"object"}]}`,
		`{"allOf":[{"type":"object","minProperties":1},{"type":"object"}]}`,
		`{"allOf":[{"type":"object","nullable":true},{"properties":{}}]}`,
		`{"allOf":[{"additionalProperties":{"type":"string"}},{"properties":{}}]}`,
		`{"allOf":[{"oneOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"b":{"type":"string"}}}]},{"properties":{}}]}`,
	} {
		rq := request(schemas(`{"Value":`+terminal+`,"Record":{"properties":{"value":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Value"}}}}}`), "Dto")
		rq.Options.Validation = true
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("nonraw composition acquired JSON equality", terminal, err)
		}
	}
	// No new inline raw declaration/source role is introduced.
	rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":{"allOf":[{"type":"object"},{"properties":{}}]}}`), "Dto")
	rq.Options.Validation = true
	if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
		t.Fatal("inline composition acquired named raw authority", err)
	}
}
