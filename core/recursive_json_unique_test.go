package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestJSONUniqueRecursiveCoverSources(t *testing.T) {
	// Preserve the exact original self-cover rejection as an opt-in graduation.
	rq := request(schemas(`{"Value":`+coverChainRoot("Value", "Value")+`,"Record":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Value"},"uniqueItems":true}}}}`), "Dto")
	rq.Options.Validation = true
	if _, err := core.Generate(rq); err != nil {
		t.Fatal(err)
	}
	for _, catalog := range []string{
		`"Value":` + coverChainRoot("Value", "Value"),
		`"Value":` + coverChainRoot("Other", "Other") + `,"Other":` + coverChainRoot("Value", "Value"),
		`"Value":` + coverChainRoot("ToOther", "Other") + `,"ToOther":{"$ref":"#/components/schemas/Other"},"Other":` + coverChainRoot("ToValue", "Value") + `,"ToValue":{"$ref":"#/components/schemas/Value"}`,
		`"Value":` + coverChainRoot("Other", "End") + `,"Other":` + coverChainRoot("Value", "End") + `,"End":` + coverChainRoot("End", "End"),
	} {
		for _, ref := range []string{"Value", "Alias", "Hops/properties/ref", "Bridge"} {
			doc := schemas(`{` + catalog + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Hops":{"properties":{"ref":{"$ref":"#/components/schemas/Alias","nullable":false}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/ref"},"Record":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/` + ref + `"},"uniqueItems":true}}}}`)
			rq := request(doc, "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(ref, err)
			}
			if !strings.Contains(source(t, got, "RecordDto"), "`openapi-kt unique JSON items`(value)") {
				t.Fatal("cover uniqueness lost", ref)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("cover uniqueness source/output drift", ref, err)
			}
			rq.Options.Validation = false
			if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("cover uniqueness accepted without opt-in", ref, err)
			}
		}
	}
}

func TestJSONUniqueRecursiveCoverCheckedBudgets(t *testing.T) {
	enumUniqueCheckedBudgets(t, coverChainRoot("Code", "Code"))
}

func TestJSONUniqueRecursiveCoverNoDropAndNoop(t *testing.T) {
	jsonUniqueReferencedNoDropAndNoop(t, `"Value":`+coverChainRoot("Value", "Value")+`, `)
}

func TestJSONUniqueRecursiveCoverExcludedSources(t *testing.T) {
	original := coverChainRoot("Value", "Value")
	for _, terminal := range []string{
		strings.Replace(original, `{"type":"string"}`, `{"type":"string","minLength":1}`, 1),
		strings.Replace(original, `{"type":"boolean"}`, `{"type":"string"}`, 1),
		strings.Replace(original, `"items":{"$ref":"#/components/schemas/Value"}`, `"items":{"$ref":"#/components/schemas/Value","nullable":true}`, 1),
		`{"$ref":"#/components/schemas/Cycle"}`,
	} {
		rq := request(schemas(`{"Value":`+terminal+`,"Cycle":{"$ref":"#/components/schemas/Value"},"Record":{"properties":{"value":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Value"}}}}}`), "Dto")
		rq.Options.Validation = true
		if terminal == strings.Replace(original, `"items":{"$ref":"#/components/schemas/Value"}`, `"items":{"$ref":"#/components/schemas/Value","nullable":true}`, 1) {
			got, e := core.GenerateWithDiagnostics(rq)
			if e != nil || len(got.Artifacts.Files) == 0 {
				t.Fatal("original complete nullable-edge cover lost JSON equality", e)
			}
			rq.Options.Validation = false
			if unchecked, e := core.GenerateWithDiagnostics(rq); e == nil || len(unchecked.Artifacts.Files) != 0 {
				t.Fatal("nullable edge disabled uniqueness opt-in", e)
			}
			continue
		}
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("incomplete/constrained/ref-only graph acquired equality", terminal, err)
		}
	}
	rq := request(schemas(`{"Value":`+original+`,"Record":{"properties":{"value":{"type":"array","uniqueItems":true,"items":`+original+`}}}}`), "Dto")
	rq.Options.Validation = true
	if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
		t.Fatal("inline cover acquired named graph authority", err)
	}
}
