package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestJSONUniqueDirectValueArrays(t *testing.T) {
	for _, items := range []string{`true`, `{}`, `{"type":"object"}`, `{"additionalProperties":true}`} {
		rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":`+items+`}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(items, err)
		}
		text := source(t, got, "RecordDto")
		if !strings.Contains(text, "`openapi-kt unique JSON items`(value)") || !strings.Contains(text, "private fun `openapi-kt JSON equality key`") {
			t.Fatal("JSON uniqueness lost", text)
		}
	}
}

func TestJSONUniqueReferencedRawTrueArrays(t *testing.T) {
	// Retain the original rejected input as an explicit graduation.
	rq := request(schemas(`{"Raw":true,"Record":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Raw"},"uniqueItems":true}}}}`), "Dto")
	rq.Options.Validation = true
	if _, err := core.Generate(rq); err != nil {
		t.Fatal(err)
	}
	for _, terminal := range []string{`true`, `{"type":"object"}`, `{"type":"object","nullable":true}`} {
		for _, ref := range []string{"Value", "Alias", "Bridge", "Hops/properties/ref"} {
			doc := schemas(`{"Value":` + terminal + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Hops":{"properties":{"ref":{"$ref":"#/components/schemas/Alias","nullable":false}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/ref"},"Record":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/` + ref + `"},"uniqueItems":true}}}}`)
			rq := request(doc, "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(terminal, ref, err)
			}
			again, repeatErr := core.GenerateWithDiagnostics(rq)
			if repeatErr != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("reference source/output drift", terminal, ref, repeatErr)
			}
			rq.Options.Validation = false
			if unchecked, e := core.GenerateWithDiagnostics(rq); e == nil || len(unchecked.Artifacts.Files) != 0 {
				t.Fatal("missing validation opt-in", terminal, ref, e)
			}
			if !strings.Contains(source(t, got, "RecordDto"), "`openapi-kt unique JSON items`(value)") {
				t.Fatal("referenced JSON check lost", terminal, ref)
			}
		}
	}
	for _, terminal := range []string{`{"type":"object","properties":{"v":{"type":"string"}}}`} {
		rq := request(schemas(`{"Value":`+terminal+`,"Record":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Value"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !strings.Contains(source(t, got, "RecordDto"), "List<org.example.models.models.ValueDto>") || strings.Contains(source(t, got, "RecordDto"), "`openapi-kt unique JSON items`") {
			t.Fatal("originalordinaryDTO nowrequires mappedDTO equality", e)
		}
	}
}

func TestJSONUniqueReferencedBudgets(t *testing.T) {
	// These shared probes exercise checked-container depth/slots and cached
	// source-hop height independently of the leaf equality policy.
	for _, terminal := range []string{`true`, `{"type":"object","nullable":true}`} {
		enumUniqueCheckedBudgets(t, terminal)
		enumUniqueSourceCacheBudgets(t, terminal)
	}
}

func TestJSONUniqueReferencedNoDropAndNoop(t *testing.T) {
	jsonUniqueReferencedNoDropAndNoop(t, `"Value":true,`)
}

func jsonUniqueReferencedNoDropAndNoop(t *testing.T, prefix string) {
	t.Helper()
	shape := `{"type":"array","items":{"$ref":"#/components/schemas/Value"},"uniqueItems":true}`
	aliasFailure(t, schemas(`{`+prefix+`"Record":{"properties":{"v":{"type":"string"}},"additionalProperties":`+shape+`}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/additionalProperties/uniqueItems")
	aliasFailure(t, schemas(`{`+prefix+`"Record":{"allOf":[{"properties":{"v":`+shape+`}}],"properties":{"v":{"type":"string"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/allOf/0/properties/v/uniqueItems")
	noop := request(schemas(`{`+prefix+`"Record":{"properties":{"value":`+strings.Replace(shape, `true`, `false`, 1)+`}}}`), "Dto")
	plain := noop
	plain.Document = []byte(strings.Replace(string(noop.Document), `,"uniqueItems":false`, "", 1))
	noop.Options.Validation, plain.Options.Validation = true, true
	first, e1 := core.GenerateWithDiagnostics(noop)
	second, e2 := core.GenerateWithDiagnostics(plain)
	if e1 != nil || e2 != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("false changed referenced JSON output", e1, e2)
	}
}
